package main

import (
	"fmt"
	"image"
)

/*
 *  This file is part of CD+Graphics Magic.
 *
 *  CD+Graphics Magic is free software: you can redistribute it and/or
 *  modify it under the terms of the GNU General Public License as
 *  published by the Free Software Foundation, either version 2 of the
 *  License, or (at your option) any later version.
 *
 *  CD+Graphics Magic is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with CD+Graphics Magic. If not, see <http://www.gnu.org/licenses/>.
 *
 */

/*
 *  This class instantiates an HTML5/Canvas CD+Graphics decoder object.
 *
 *  This is the "low resource" version, and should be very close
 *  to as fast as possible with JavaScript.
 *
 *  The difference between the "low resource" and normal version
 *  is that this one packs each 6 pixel font line in to one
 *  array value, unrolling some loops and minimizing array lookups.
 *
 *  The only concession made is lack of H/V "offset" support used
 *  for smooth scrolling.
 *  Block based scrolls *are* still supported, however, so the basic
 *  intent of the graphics is presented, but less than ideally.
 *
 *  It is recommended for CPU constrained (eg. mobile or embedded) devices.
 *
 */

// Useful enums for CD+Graphics...

const (
	VRAM_SIZE       = 300 * 216 // Total linear size of VRAM, in pixels.
	VRAM_WIDTH      = 300       // Width (or pitch) of VRAM, in pixels.
	VRAM_HEIGHT     = 216       // Height of VRAM, in pixels.
	VISIBLE_SIZE    = 288 * 192 // Total linear size of visible screen, in pixels.
	VISIBLE_WIDTH   = 288       // Width (or pitch) of visible screen, in pixels.
	VISIBLE_HEIGHT  = 192       // Height of visible screen, in pixels.
	FONT_WIDTH      = 6         // Width of  one "font" (or block).
	FONT_HEIGHT     = 12        // Height of one "font" (or block).
	NUM_X_FONTS     = 50        // Number of horizontal fonts contained in VRAM.
	NUM_Y_FONTS     = 18        // Number of vertical fonts contained in VRAM.
	PALETTE_ENTRIES = 16        // Number of CLUT palette entries.
	TV_GRAPHICS     = 0x09      // 50x18 (48x16) 16 color TV graphics mode.
	MEMORY_PRESET   = 0x01      // Set all VRAM to palette index.
	BORDER_PRESET   = 0x02      // Set border to palette index.
	LOAD_CLUT_LO    = 0x1E      // Load Color Look Up Table index 0 through 7.
	LOAD_CLUT_HI    = 0x1F      // Load Color Look Up Table index 8 through 15.
	COPY_FONT       = 0x06      // Copy 12x6 pixel font to screen.
	XOR_FONT        = 0x26      // XOR 12x6 pixel font with existing VRAM values.
	SCROLL_PRESET   = 0x14      // Update scroll offset, copying if 0x20 or 0x10.
	SCROLL_COPY     = 0x18      // Update scroll offset, setting color if 0x20 or 0x10.
)

var (
	internal_palette        = make([]byte, PALETTE_ENTRIES)
	internal_vram           = make([]byte, NUM_X_FONTS*VRAM_HEIGHT)
	internal_dirty_blocks   = make([]byte, 900)
	internal_rgba_context   = image.NewRGBA(image.Rect(0, 0, VISIBLE_WIDTH, VISIBLE_HEIGHT))
	internal_rgba_imagedata = internal_rgba_imagedata.Pix
	internal_usedirtyrect   = true

	internal_border_index = 0x00 // The current border palette index.
	internal_current_pack = 0x00 //

	internal_border_dirty = false
	internal_screen_dirty = false
)

func main() {
	fmt.Println("Compiles baby!")
}

func resetCDGState() {
	internal_current_pack = 0x00
	internal_border_index = 0x00
	clearPalette()
	clearVRAM(0x00)
	clearDirtyBlocks()
}

func clearPalette() {
	for idx := 0; idx < PALETTE_ENTRIES; idx++ {
		internal_palette[idx] = 0x00
	}
}

func get_current_pack() byte {
	//casting: must test!!!
	return byte(internal_current_pack)
}

/* Possibly not needed!
func set_dirtyrect(requested_value) {
	internal_usedirtyrect = requested_value
}
*/

func redrawCanvas() {

	if internal_border_dirty || internal_screen_dirty {
		// render_screen_to_rgb()
		internal_screen_dirty = false
		clearDirtyBlocks()
		// internal_rgba_context.putImageData(internal_rgba_imagedata, 0, 0)
	} else {
		var local_context = internal_rgba_context
		var local_rgba_imagedata = internal_rgba_imagedata

		update_needed := false
		var blk = 0x00

		//NOTE: test the post-increment (Go does not have pre, so had to change it)

		for y_blk := 1; y_blk <= 16; y_blk++ {

			blk = y_blk*NUM_X_FONTS + 1

			for x_blk := 1; x_blk <= 48; x_blk++ {

				if internal_dirty_blocks[blk] {

					render_block_to_rgb(x_blk, y_blk)

					if internal_usedirtyrect {
						local_context.putImageData(local_rgba_imagedata, 0, 0,
							(x_blk-1)*FONT_WIDTH,
							(y_blk-1)*FONT_HEIGHT,
							FONT_WIDTH,
							FONT_HEIGHT)
					} else {
						update_needed = true
					}

					internal_dirty_blocks[blk] = 0x00
				}
				//Note: test the post-increment
				blk++
			}
		}
		// Update the whole screen for browsers where dirty rect isn't supported.
		// Since this can't be detected(???) in any way, it has to be User Agent selected, or an actual user option.
		// TODO: See if a dirty rect-based partial update of known pixel values combined with a getImageData
		//       call could be used to determine if it works correctly *without* evil browser sniffing.
		if update_needed {
			//local_context.putImageData(local_rgba_imagedata, 0, 0);
		}
	}
}

// Decode to pack playback_position, using cdg_file_data.
func decode_packs(cdg_file_data []byte, playback_position int) {

	for curr_pack := internal_current_pack; curr_pack < playback_position; curr_pack++ {

		start_offset := curr_pack * 24
		curr_command := cdg_file_data[start_offset] & 0x3F

		if curr_command == TV_GRAPHICS {
			// Slice the file array down to a single pack array.
			this_pack := cdg_file_data[start_offset : start_offset+24]
			// Pluck out the graphics instruction.
			curr_instruction := this_pack[1] & 0x3F
			// Perform the instruction action.
			switch curr_instruction {
			case MEMORY_PRESET:
				proc_MEMORY_PRESET(this_pack)

			case BORDER_PRESET:
				proc_BORDER_PRESET(this_pack)

			case LOAD_CLUT_LO:
			case LOAD_CLUT_HI:
				proc_LOAD_CLUT(this_pack)

			case COPY_FONT:
			case XOR_FONT:
				proc_WRITE_FONT(this_pack)

			case SCROLL_PRESET:
			case SCROLL_COPY:
				proc_DO_SCROLL(this_pack)

			}
		}
	}
	internal_current_pack = playback_position
}

func fill_line_with_palette_index(requested_index byte) byte {

	adjusted_value := requested_index          // Pixel 0
	adjusted_value |= (requested_index << 004) // Pixel 1
	adjusted_value |= (requested_index << 010) // Pixel 2
	adjusted_value |= (requested_index << 014) // Pixel 3
	adjusted_value |= (requested_index << 020) // Pixel 4
	adjusted_value |= (requested_index << 024) // Pixel 5

	return adjusted_value
}

func clearDirtyBlocks() {
	for blk := 0; blk < 900; blk++ {
		internal_dirty_blocks[blk] = 0x00
	}
}

func clearVRAM(colorIndex byte) {

	packed_line_value := fill_line_with_palette_index(colorIndex)

	for pxl := 0; pxl < len(internal_vram); pxl++ {
		internal_vram[pxl] = packed_line_value
	}

	internal_screen_dirty = true
}

func render_screen_to_rgb() {

	vis_width := 48
	vis_height := VISIBLE_HEIGHT

	vram_loc := 601           // Offset into VRAM array.
	rgb_loc := 0x00           // Offset into RGBA array.
	curr_rgb := 0x00          // RGBA value of current pixel.
	curr_line_indices := 0x00 // Packed font row index values.

	for y_pxl := 0; y_pxl < vis_height; y_pxl++ {
		for x_pxl := 0; x_pxl < vis_width; x_pxl++ {

			//for the Go version, maybe don't have to unroll the loop cause it's getting ugly.
			//NOTE: these values are shifted by Octal numbers looks like ie: 010
			//NOTE: In Go, ++ is a statement not expression, so had to post-increment after-the-fact

			curr_line_indices = internal_vram[vram_loc] // Get the current line segment indices.
			vram_loc++
			curr_rgb = internal_palette[(curr_line_indices>>000)&0x0F]  // Get the RGB value for pixel 0.
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 0.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 0.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 0.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF
			rgb_loc++ // Set alpha value (fully opaque) for pixel 0.

			curr_rgb = internal_palette[(curr_line_indices>>004)&0x0F]  // Get the RGB value for pixel 1.
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 1.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 1.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 1.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF // Set alpha value (fully opaque) for pixel 1.
			rgb_loc++
			curr_rgb = internal_palette[(curr_line_indices>>010)&0x0F] // Get the RGB value for pixel 2.

			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 2.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 2.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 2.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF
			rgb_loc++ // Set alpha value (fully opaque) for pixel 2.

			curr_rgb = internal_palette[(curr_line_indices>>014)&0x0F]  // Get the RGB value for pixel 3.
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 3.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 3.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 3.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF // Set alpha value (fully opaque) for pixel 3.
			rgb_loc++

			curr_rgb = internal_palette[(curr_line_indices>>020)&0x0F]  // Get the RGB value for pixel 4.
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 4.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 4.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 4.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF
			rgb_loc++ // Set alpha value (fully opaque) for pixel 4.

			curr_rgb = internal_palette[(curr_line_indices>>024)&0x0F]  // Get the RGB value for pixel 5.
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 5.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 5.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 5.
			rgb_loc++
			internal_rgba_imagedata[rgb_loc] = 0xFF // Set alpha value (fully opaque) for pixel 5.
			rgb_loc++
			// Or, instead, index 0 could be set transparent to show background image/video.
			// Alternately, SET_TRANSPARENT instruction could be implemented to set 6bit transparency.
			// Unfortunately, I don't think many (any?) discs bother to set it :-/...
		}
		vram_loc += 2 // Skip the offscreen font blocks.
	}
}

func render_block_to_rgb(x_start, y_start int) {
	vram_loc := (y_start * NUM_X_FONTS * FONT_HEIGHT) + x_start // Offset into VRAM array.
	vram_inc := NUM_X_FONTS
	vram_end := vram_loc + (NUM_X_FONTS * FONT_HEIGHT)     // VRAM location to end.
	rgb_loc := (y_start - 1) * FONT_HEIGHT * VISIBLE_WIDTH // Row start.
	rgb_loc += (x_start - 1) * FONT_WIDTH                  // Column start
	rgb_loc *= 4                                           // RGBA, 1 pxl = 4 bytes.

	rgb_inc := (VISIBLE_WIDTH - FONT_WIDTH) * 4
	curr_rgb := 0x00          // RGBA value of current pixel.
	curr_line_indices := 0x00 // Packed font row index values.

	for vram_loc < vram_end {
		curr_line_indices = internal_vram[vram_loc]                 // Get the current line segment indices.
		curr_rgb = internal_palette[(curr_line_indices>>000)&0x0F]  // Get the RGB value for pixel 0.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 0.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 0.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 0.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF
		rgb_loc++                                                   // Set alpha value (fully opaque) for pixel 0.
		curr_rgb = internal_palette[(curr_line_indices>>004)&0x0F]  // Get the RGB value for pixel 1.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 1.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 1.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 1.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF
		rgb_loc++                                                   // Set alpha value (fully opaque) for pixel 1.
		curr_rgb = internal_palette[(curr_line_indices>>010)&0x0F]  // Get the RGB value for pixel 2.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 2.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 2.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 2.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF
		rgb_loc++                                                   // Set alpha value (fully opaque) for pixel 2.
		curr_rgb = internal_palette[(curr_line_indices>>014)&0x0F]  // Get the RGB value for pixel 3.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 3.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 3.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 3.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF
		rgb_loc++                                                   // Set alpha value (fully opaque) for pixel 3.
		curr_rgb = internal_palette[(curr_line_indices>>020)&0x0F]  // Get the RGB value for pixel 4.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 4.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 4.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 4.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF
		rgb_loc++                                                   // Set alpha value (fully opaque) for pixel 4.
		curr_rgb = internal_palette[(curr_line_indices>>024)&0x0F]  // Get the RGB value for pixel 5.
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 020) & 0xFF // Set red value for pixel 5.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 010) & 0xFF // Set green value for pixel 5.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = (curr_rgb >> 000) & 0xFF // Set blue value for pixel 5.
		rgb_loc++
		internal_rgba_imagedata[rgb_loc] = 0xFF // Set alpha value (fully opaque) for pixel 5.
		rgb_loc++
		// Or, instead, index 0 could be set transparent to show background image/video.
		// Alternately, SET_TRANSPARENT instruction could be implemented to set 6bit transparency.
		// Unfortunately, I don't think many (any?) discs bother to set it :-/...
		vram_loc += vram_inc // Move to the first column of the next row of this font block in VRAM.
		rgb_loc += rgb_inc   // Move to the first column of the next row of this font block in RGB pixels.
	}
}

//########## PRIVATE GRAPHICS DECODE FUNCTIONS ##########//

func proc_BORDER_PRESET(cdg_pack []byte) {
	// NOTE: The "border" is actually a DIV element, which can be very expensive to change in some browsers.
	// This somewhat bizarre check ensures that the DIV is only touched if the actual RGB color is different,
	// but the border index variable is always set... A similar check is also performed during palette update.
	new_border_index := cdg_pack[4] & 0x3F // Get the border index from subcode.
	// Check if the new border **RGB** color is different from the old one.
	if internal_palette[new_border_index] != internal_palette[internal_border_index] {
		internal_border_dirty = true // Border needs updating.
	}

	internal_border_index = new_border_index // Set the new index.
}

func proc_MEMORY_PRESET(cdg_pack []byte) {
	clearVRAM(cdg_pack[4] & 0x3F)
}

func proc_LOAD_CLUT(cdg_pack []byte) {

	// If instruction is 0x1E then 8*0=0, if 0x1F then 8*1=8 for offset.
	pal_offset := (cdg_pack[1] & 0x01) * 8
	// Step through the eight color indices, setting the RGB values.
	for pal_inc := 0; pal_inc < 8; pal_inc++ {
		temp_idx := pal_inc + pal_offset
		temp_rgb := 0x00000000
		temp_entry := 0x00000000
		// Set red.
		temp_entry = (cdg_pack[pal_inc*2+4] & 0x3C) >> 2
		temp_rgb |= (temp_entry * 17) << 020
		// Set green.
		temp_entry = ((cdg_pack[pal_inc*2+4] & 0x03) << 2) | ((cdg_pack[pal_inc*2+5] & 0x30) >> 4)
		temp_rgb |= (temp_entry * 17) << 010
		// Set blue.
		temp_entry = cdg_pack[pal_inc*2+5] & 0x0F
		temp_rgb |= (temp_entry * 17) << 000

		// Put the full RGB value into the index position, but only if it's different.
		if temp_rgb != innternal_palette[temp_idx] {
			innternal_palette[temp_idx] = temp_rgb
			internal_screen_dirty = true // The colors are now different, so we need to update the whole screen.

			if temp_idx == internal_border_index {
				internal_border_dirty = true
			} // The border color has changed.
		}
	}
}

func proc_WRITE_FONT(cdg_pack []byte) {
	var local_vram = internal_vram
	var local_dirty = internal_dirty_blocks
	// Hacky hack to play channels 0 and 1 only... Ideally, there should be a function and user option to get/set.
	active_channels := 0x03
	// First, get the channel...
	subcode_channel := ((cdg_pack[4] & 0x30) >> 2) | ((cdg_pack[5] & 0x30) >> 4)
	xor_var := cdg_pack[1] & 0x20
	// Then see if we should display it.
	if (active_channels >> subcode_channel) & 0x01 {
		x_location := cdg_pack[7] & 0x3F // Get horizontal font location.
		y_location := cdg_pack[6] & 0x1F // Get vertical font location.

		// Verify we're not going to overrun the boundaries (i.e. bad data from a scratched disc).
		if (x_location <= 49) && (y_location <= 17) {
			start_pixel := y_location*600 + x_location // Location of first pixel of this font in linear VRAM.
			// NOTE: Profiling indicates charCodeAt() uses ~80% of the CPU consumed for this function.
			// Caching these values reduces that to a negligible amount.

			current_indexes = make([]byte, 2)
			current_indexes[0] = (cdg_pack[4] & 0x0F)
			current_indexes[1] = (cdg_pack[5] & 0x0F)

			current_row := 0x00 // Subcode byte for current pixel row.
			temp_pxl := 0x00    // Decoded and packed 4bit pixel index values of current row.
			for y_inc := 0; y_inc < 12; y_inc++ {
				var pix_pos = y_inc*50 + start_pixel // Location of the first pixel of this row in linear VRAM.
				current_row = cdg_pack[y_inc+8]      // Get the subcode byte for the current row.
				temp_pxl = (current_indexes[(current_row>>5)&0x01] << 000)
				temp_pxl |= (current_indexes[(current_row>>4)&0x01] << 004)
				temp_pxl |= (current_indexes[(current_row>>3)&0x01] << 010)
				temp_pxl |= (current_indexes[(current_row>>2)&0x01] << 014)
				temp_pxl |= (current_indexes[(current_row>>1)&0x01] << 020)
				temp_pxl |= (current_indexes[(current_row>>0)&0x01] << 024)

				//NOTE: figure out truthy-ness of xor_var
				if xor_var != 0 {
					local_vram[pix_pos] ^= temp_pxl
				} else {
					local_vram[pix_pos] = temp_pxl
				}
			} // End of Y loop.
			// Mark this block as needing an update.
			local_dirty[y_location*50+x_location] = 0x01
		} // End of location check.
	} // End of channel check.
}

func proc_DO_SCROLL(cdg_pack []byte) {
	direction := 0                        // H/V direction flag.
	copy_flag = (cdg_pack[1] & 0x08) >> 3 // Type of copy (memory preset or copy).
	color = cdg_pack[4] & 0x0F            // Color index to use for preset type.

	//TODOD: check what value of direction is
	// Process horizontal commands.
	if direction = (cdg_pack[5] & 0x30) >> 4; direction != 0 {
		proc_VRAM_HSCROLL(direction, copy_flag, color)
	}

	// Process vertical commands.
	if direction = (cdg_pack[6] & 0x30) >> 4; direction != 0 {
		proc_VRAM_VSCROLL(direction, copy_flag, color)
	}

	internal_screen_dirty = true // Entire screen needs to be redrawn.
}

//TODO: proc_VRAM_HSCROLL
//TODO: proc_VRAM_VSCROLL
