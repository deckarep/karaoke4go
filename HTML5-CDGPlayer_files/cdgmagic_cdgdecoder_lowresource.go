package main

import "fmt"

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
	internal_palette      = make([]byte, PALETTE_ENTRIES)
	internal_vram         = make([]byte, NUM_X_FONTS*VRAM_HEIGHT)
	internal_dirty_blocks = make([]byte, 900)

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
		// internal_screen_dirty = 0
		// clear_dirty_blocks()
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

					if internal_usedirtyrect == 0x01 {
						local_context.putImageData(local_rgba_imagedata, 0, 0,
							(x_blk-1)*FONT_WIDTH,
							(y_blk-1)*FONT_HEIGHT,
							FONT_WIDTH,
							FONT_HEIGHT)
					} else {
						update_needed = 0x01
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

func clearVRAM(colorIndex byte) {

	packed_line_value := fill_line_with_palette_index(colorIndex)

	for pxl := 0; pxl < len(internal_vram); pxl++ {
		internal_vram[pxl] = packed_line_value
	}

	internal_screen_dirty = true
}

func clearDirtyBlocks() {
	for blk := 0; blk < 900; blk++ {
		internal_dirty_blocks[blk] = 0x00
	}
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
