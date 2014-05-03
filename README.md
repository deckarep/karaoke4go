karaoke4go
===============

karaoke4go is an extremely *early* alpha prototype of the CDG file format that was developed by Phillips and Sony entertainment to display timed lyrics along with music for the purpose of getting drunk with your friends and singing horribly into a mic.

It was delivered on audio Compact Discs and took advantage of subcode channels R through W where 16-color (4-bit) graphics could be rendered onto a 300x216 pixel size bitmap.

## screenshots

![Title Screen](../master/screenshots/title.png?raw=true)

![Key Screen](../master/screenshots/key.png?raw=true)

![Intro Screen](../master/screenshots/intro.png?raw=true)


## why?

Why not? Go is a fantastic language and this was an excuse to have some fun with it while exploring the image packages.  Unfortunately, Go doesn't have a native package for rendering GUIs so this package currently only renders a series of .png images.  Eventually, the idea is to utilize some community driven package such as GoQML to render the graphics in realtime so this thing actually works.

## a partial port

So, I've actually built this from scratch twice before once using raw C and the popular SDL library.  And the second time, I built this in Flash way back when the raw bitmap api was introduced.  This time around, I did get a little lazy and actually ported this version from the excellent: CDGMagic HTML5 canvas based version located at: http://cdgmagic.sourceforge.net/html5_cdgplayer/  This version actually works beautifully and runs smooth.  Again, consider this version a fun excercize in Go...at least for now.

## caveats


* It's not a typical Go package yet, so you can't import it
* There are currently no tests
* The code in its current state is partially broken, but it does render mostly correct at this point
* The code eventually should be cleaned up and simplified with more idiomatic Go code
* It does not play in realtime, it only generates an image sequence currently
* There is no music implementation yet
* It has not been optimized yet

## contributions

* are encouraged



