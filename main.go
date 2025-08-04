package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gonutz/prototype/draw"
)

//go:embed rsc
var rsc embed.FS

func main() {
	draw.OpenFile = func(path string) (io.ReadCloser, error) {
		return rsc.Open(path)
	}

	animationFrames := []string{
		"rsc/arms_center.png",
		"rsc/arms_up.png",
		"rsc/arms_center.png",
		"rsc/arms_down.png",
	}
	// At most one item from each accessory group is picked.
	accessoryGroups := [][]string{
		{"hat"},
		{"tie", "bowtie", "shirt"},
		{"round_glasses", "square_glasses", "sunglasses"},
		{"earring"},
	}
	backgroundImages := []string{
		"rsc/city0.png",
		"rsc/city1.png",
	}
	backgroundColor := rgb(151, 255, 255)
	const (
		windowW, windowH           = 1500, 800
		tailDownImage              = "rsc/tail_down.png"
		tailCenterImage            = "rsc/tail_center.png"
		tailUpImage                = "rsc/tail_up.png"
		deadFrame                  = "rsc/dead.png"
		bumpFrame                  = "rsc/bump.png"
		pipeImage                  = "rsc/pipe.png"
		cloudImage                 = "rsc/cloud.png"
		gopherSpeed                = 5
		gravity                    = 0.5
		gapHeight                  = 300
		firstGapX                  = 1300
		gapDistX                   = 600
		finalGopherX               = 100
		gopherCollisionRadius      = 50
		clickYSpeed                = -14.0
		minVisiblePipeHeight       = 80
		pipeShakeFrameCount        = 40
		musicIntroFile             = "rsc/music_intro.wav"
		musicIntroLengthInSeconds  = 6
		musicLoopFile              = "rsc/music_loop.wav"
		musicLoopLengthInSeconds   = 14
		deceasedTextFadeFrameCount = 60
		cursorHideTimeout          = 120
	)

	var (
		animationIndex      int
		nextFlapIn          int
		gopherXOffset       int
		x                   float64
		y                   float64
		xSpeed              float64
		ySpeed              float64
		rotation            float64
		targetRotation      float64
		isAlive             bool
		gaps                [10]gap
		nextGapX            int
		score               int
		scoreAnimationTime  float64
		restartableTime     int
		backgroundTiles     []backgroundTile
		highscore           int
		needToPlayFlapSound bool
		flapSoundCoolDown   int
		playDeathSoundIn    int
		bumpOnHead          bool
		clouds              [6]cloud
		accessories         []string
		// killCount is not always the same as len(killHistory). When we kill
		// the latest gopher, we add it to the killHistory right away, but we
		// wait for the restart screen until we update the kill count in the
		// bottom right hand corner.
		killCount          int
		killHistory        []kill
		wasRestartable     bool
		name               string
		nameAlpha          float32
		nameAnimationTime  int
		deceasedTextTime   int
		hideCursorInFrames int
	)

	randomGapY := func() int {
		top := gapHeight/2 + minVisiblePipeHeight
		bottom := windowH - gapHeight/2 - minVisiblePipeHeight
		return top + rand.Intn(bottom-top)
	}

	randomCityImage := func() string {
		i := rand.Intn(len(backgroundImages))
		return backgroundImages[i]
	}

	randomCloudScale := func() float64 {
		return 0.5 + rand.Float64()*0.5
	}

	randomCloudY := func() int {
		minY := -100
		maxY := 360
		return minY + rand.Intn(maxY-minY)
	}

	randomName := func() string {
		// TODO Use up all names before repeating a name.
		i := rand.Intn(len(nameList))
		return nameList[i]
	}

	restart := func() {
		animationIndex = 0
		nextFlapIn = 0
		gopherXOffset = -finalGopherX - 150
		x = 0.0
		y = 400.0
		xSpeed = 0.0
		ySpeed = clickYSpeed
		rotation = 0.0
		targetRotation = 0.0
		isAlive = true
		nextGapX = firstGapX
		for i := range gaps {
			gaps[i] = gap{}
			gaps[i].centerX = nextGapX
			gaps[i].centerY = randomGapY()
			nextGapX += gapDistX
		}
		score = 0
		scoreAnimationTime = 0.0
		restartableTime = 0
		backgroundTiles = backgroundTiles[:0]
		killHistory = loadKillHistory()
		killCount = len(killHistory)
		highscore = 0
		for _, k := range killHistory {
			highscore = max(highscore, k.Score)
		}
		needToPlayFlapSound = true
		playDeathSoundIn = 0
		bumpOnHead = false
		for i := range clouds {
			clouds[i].scale = randomCloudScale()
			clouds[i].x = float64(-350 + rand.Intn(windowW+350))
			clouds[i].y = randomCloudY()
		}

		lastAccessories := slices.Clone(accessories)
		for slices.Equal(lastAccessories, accessories) {
			accessories = accessories[:0]
			accessoryChance := 0.33
			for _, group := range accessoryGroups {
				if rand.Float64() < accessoryChance {
					i := rand.Intn(len(group))
					accessories = append(accessories, group[i])
				}
			}
		}
		wasRestartable = false
		name = randomName()
		nameAlpha = 1.0
		nameAnimationTime = 0
		deceasedTextTime = deceasedTextFadeFrameCount
		hideCursorInFrames = cursorHideTimeout
	}
	restart()

	preloadImage := func(window draw.Window) bool {
		preloaded := true

		preload := func(img string) {
			_, _, err := window.ImageSize(img)

			if err == draw.ErrImageLoading {
				preloaded = false
			} else if err != nil {
				panic(err)
			}
		}

		files, _ := rsc.ReadDir("rsc")
		for _, file := range files {
			path := "rsc/" + file.Name()
			if strings.HasSuffix(path, ".png") {
				preload(path)
			}
		}

		return preloaded
	}

	imagesAreLoaded := false
	var nextMusicStart time.Time
	var lastMouseX, lastMouseY int

	draw.RunWindow("Flappy Go", windowW, windowH, func(window draw.Window) {
		window.SetIcon("rsc/icon.png")

		if !imagesAreLoaded {
			imagesAreLoaded = preloadImage(window)
			if !imagesAreLoaded {
				window.DrawText("Loading images...", 0, 0, draw.White)
				return
			}
		}

		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}
		window.BlurImages(true)

		if nextMusicStart.IsZero() {
			window.PlaySoundFile(musicIntroFile)
			nextMusicStart = time.Now().Add(seconds(musicIntroLengthInSeconds))
		}

		now := time.Now()
		if now.Equal(nextMusicStart) || now.After(nextMusicStart) {
			window.PlaySoundFile(musicLoopFile)
			nextMusicStart = now.Add(seconds(musicLoopLengthInSeconds))
		}

		pipeW, pipeH, _ := window.ImageSize(pipeImage)

		restartable := y > 3*windowH

		if restartable != wasRestartable {
			killCount++
			wasRestartable = restartable
		}

		// Update game state.
		clickedWithMouse := len(window.Clicks()) > 0
		clicked := clickedWithMouse ||
			len(window.Characters()) > 0 ||
			window.WasKeyPressed(draw.KeyUp) ||
			window.WasKeyPressed(draw.KeyEnter) ||
			window.WasKeyPressed(draw.KeyNumEnter)

		if isAlive {
			hideCursorInFrames--
		}
		mouseX, mouseY := window.MousePosition()
		if clickedWithMouse ||
			mouseX != lastMouseX || mouseY != lastMouseY {
			hideCursorInFrames = cursorHideTimeout
		}
		lastMouseX, lastMouseY = mouseX, mouseY

		window.ShowCursor(hideCursorInFrames > 0)

		if restartable && clicked {
			restart()
			clicked = false
		}

		if isAlive && clicked {
			ySpeed = clickYSpeed
			nextFlapIn = 0
			needToPlayFlapSound = true
		}

		flapSoundCoolDown--

		if needToPlayFlapSound && flapSoundCoolDown <= 0 {
			window.PlaySoundFile("rsc/flap.wav")
			flapSoundCoolDown = 30
		}

		needToPlayFlapSound = false

		nextFlapIn--
		if nextFlapIn <= 0 {
			const (
				slowestFlapYSpeed = 10.0
				minFlapIn         = 1
				maxFlapIn         = 10
			)
			relative := (ySpeed - clickYSpeed) / (slowestFlapYSpeed - clickYSpeed)
			nextFlapIn = round(minFlapIn + relative*(maxFlapIn-minFlapIn))
			animationIndex = (animationIndex + 1) % len(animationFrames)
		}

		if gopherXOffset < finalGopherX {
			// Slide in the gopher into the screen.
			gopherXOffset = min(gopherXOffset+gopherSpeed, finalGopherX)

			if gopherXOffset == finalGopherX {
				xSpeed = gopherSpeed
			}
		}

		if gopherXOffset == finalGopherX {
			nameAlpha = max(nameAlpha-0.33/60.0, 0)
		}

		if !isAlive && xSpeed > 0 {
			xSpeed = max(0, xSpeed-0.15)
		}

		x += xSpeed
		for i := range gaps {
			if gaps[i].centerX-round(x) < -pipeW/2 {
				gaps[i] = gap{}
				gaps[i].centerX = nextGapX
				gaps[i].centerY = randomGapY()
				nextGapX += gapDistX

				score++
				window.PlaySoundFile("rsc/score.wav")

				if score > highscore {
					highscore = score
				}

				scoreAnimationTime = 1.0
			}
		}
		y += ySpeed
		ySpeed += gravity

		wasAlive := isAlive

		if isAlive && y <= -30 {
			// Drop dead on hitting the ceiling.
			isAlive = false
			ySpeed = 0
			bumpOnHead = true
			window.PlaySoundFile("rsc/hit_ceiling.wav")
			playDeathSoundIn = 30
		}
		if isAlive && y >= windowH-145 {
			// Drop dead on hitting the floor. Give it a little upward motion to
			// make the user see that it is dead.
			ySpeed = -25
			isAlive = false
			window.PlaySoundFile("rsc/hit_floor.wav")
			playDeathSoundIn = 60
		}

		// Collide with the pipes.
		gopherCollisionCircle := func() circle {
			gopherW, gopherH, _ := window.ImageSize(animationFrames[0])
			return circle{
				centerX: gopherXOffset + finalGopherX + gopherW/2,
				centerY: round(y) + gopherH/2,
				radius:  gopherCollisionRadius,
			}
		}

		topPipeCollisionRect := func(gap gap) rectangle {
			left := gap.centerX - pipeW/2 - round(x) + 5
			return rectangle{
				left:   left,
				top:    0,
				right:  left + pipeW - 10,
				bottom: gap.centerY - gapHeight/2 - 2,
			}
		}

		bottomPipeCollisionRect := func(gap gap) rectangle {
			left := gap.centerX - pipeW/2 - round(x) + 5
			return rectangle{
				left:   left,
				top:    gap.centerY + gapHeight/2 + 2,
				right:  left + pipeW - 10,
				bottom: windowH,
			}
		}

		for i := range gaps {
			gaps[i].shakeTimer--
		}

		if isAlive {
			gopher := gopherCollisionCircle()
			for i, gap := range gaps {
				top := topPipeCollisionRect(gap)
				bottom := bottomPipeCollisionRect(gap)
				topCollides := collides(gopher, top)
				bottomCollides := collides(gopher, bottom)
				if topCollides || bottomCollides {
					isAlive = false
					window.PlaySoundFile("rsc/hit_pipe.wav")
					playDeathSoundIn = 25
					gaps[i].topPipeShaking = topCollides
					gaps[i].bottomPipeShaking = bottomCollides
					gaps[i].shakeTimer = pipeShakeFrameCount
				}
			}
		}

		playDeathSoundIn--
		if playDeathSoundIn == 0 {
			window.PlaySoundFile("rsc/death.wav")
		}

		if wasAlive && !isAlive {
			killHistory = append(killHistory, kill{
				Name:        name,
				Score:       score,
				Accessories: slices.Clone(accessories),
			})
			saveKillHistory(killHistory)
		}

		targetRotation = ySpeed * 1.5
		rotation = 0.5*targetRotation + 0.5*rotation

		if scoreAnimationTime > 0 {
			scoreAnimationTime = max(0, scoreAnimationTime-0.05)
		}

		backgroundW, backgroundH, _ := window.ImageSize(backgroundImages[0])
		backgroundXDist := backgroundW - 20
		if len(backgroundTiles) == 0 {
			// Initialize the background tiles.
			for i := range 10 {
				backgroundTiles = append(backgroundTiles, backgroundTile{
					image:   randomCityImage(),
					x:       float64((i - 1) * backgroundXDist),
					yOffset: rand.Intn(150),
				})
			}
		}

		backgroundXOffset := -xSpeed * 0.333
		for i := range backgroundTiles {
			backgroundTiles[i].x += backgroundXOffset
		}
		for i := range backgroundTiles {
			if backgroundTiles[i].x < float64(-backgroundW) {
				// Respawn this tile after the last background tile.
				lastTileIndex := (len(backgroundTiles) + i - 1) % len(backgroundTiles)
				backgroundTiles[i].x = backgroundTiles[lastTileIndex].x + float64(backgroundXDist)
			}
		}

		cloudW, cloudH, _ := window.ImageSize(cloudImage)
		baseCloudSpeed := -xSpeed * 0.2
		for i := range clouds {
			clouds[i].x += baseCloudSpeed * clouds[i].scale
			if clouds[i].x < float64(-cloudW) {
				clouds[i].x = windowW
				clouds[i].scale = randomCloudScale()
				clouds[i].y = randomCloudY()
			}
		}

		nameAnimationTime++

		if !isAlive && deceasedTextTime > 0 {
			deceasedTextTime--
		}

		// Draw game.

		window.FillRect(0, 0, 9999, 9999, backgroundColor)

		for _, cloud := range clouds {
			x := round(cloud.x)
			w := round(float64(cloudW) * cloud.scale)
			h := round(float64(cloudH) * cloud.scale)
			window.DrawImageFileTo(cloudImage, x, cloud.y, w, h, 0)
		}

		for _, tile := range backgroundTiles {
			tileX := round(tile.x)
			tileY := windowH - backgroundH + tile.yOffset
			window.DrawImageFile(tile.image, tileX, tileY)
		}

		for _, gap := range gaps {
			gapX := gap.centerX - pipeW/2 - round(x)

			rotation := 0
			if gap.shakeTimer > 0 {
				amplitude := 7 * float64(gap.shakeTimer) / pipeShakeFrameCount
				t := float64(pipeShakeFrameCount - gap.shakeTimer)
				rotation = round(math.Sin(t*0.8) * amplitude)
			}

			// Bottom pipe.
			bottomRotation := 0
			if gap.bottomPipeShaking && gap.shakeTimer > 0 {
				bottomRotation = rotation
			}
			bottomY := gap.centerY + gapHeight/2
			window.DrawImageFileRotated(pipeImage, gapX, bottomY, bottomRotation)

			// Top pipe.
			topRotation := 0
			if gap.topPipeShaking && gap.shakeTimer > 0 {
				topRotation = rotation
			}
			topY := gap.centerY - gapHeight/2 - pipeH
			window.DrawImageFileRotated(pipeImage, gapX, topY, 180+topRotation)
		}

		// Render the gopher.
		gopherImage := deadFrame
		if isAlive {
			gopherImage = animationFrames[animationIndex]
		} else if bumpOnHead {
			gopherImage = bumpFrame
		}

		tail := tailCenterImage
		if ySpeed > 7 {
			tail = tailUpImage
		}
		if ySpeed < -7 {
			tail = tailDownImage
		}
		if !isAlive {
			tail = tailDownImage
		}

		gopherX, gopherY := gopherXOffset+finalGopherX, round(y)
		gopherRotation := round(rotation)
		window.DrawImageFileRotated(gopherImage, gopherX, gopherY, gopherRotation)
		window.DrawImageFileRotated(tail, gopherX, gopherY, gopherRotation)
		for _, a := range accessories {
			img := "rsc/" + a + ".png"
			window.DrawImageFileRotated(img, gopherX, gopherY, gopherRotation)
		}

		// Render the animated name above the gopher's head.
		gopherW, _, _ := window.ImageSize(gopherImage)
		const headNameScale = 4
		headNameW, headNameH := window.GetScaledTextSize(name, headNameScale)
		headNameX := gopherXOffset + finalGopherX + gopherW/2 - headNameW/2
		headNameY := gopherY - headNameH
		runeW, _ := window.GetScaledTextSize("x", headNameScale)
		runeX := headNameX
		runeI := 0
		for _, r := range name {
			yOffset := (math.Sin(0.5*float64(runeI)+0.075*float64(nameAnimationTime)) + 1) / 2
			runeY := headNameY - round(yOffset*0.75*float64(headNameH))
			window.DrawScaledText(string(r), runeX, runeY, headNameScale, draw.RGBA(0, 0, 0, nameAlpha))
			runeX += runeW
			runeI++
		}

		const (
			regularScoreScale = 7.0
			maxScoreScale     = 12.0
		)
		scoreScale := float32(regularScoreScale)
		if scoreAnimationTime > 0 {
			scoreArc := (math.Sin(1.5*math.Pi+2*math.Pi*scoreAnimationTime) + 1) * 0.5
			scoreScale = float32(regularScoreScale + scoreArc*(maxScoreScale-regularScoreScale))
		}
		scoreText := fmt.Sprintf(" %d ", score)
		scoreW, _ := window.GetScaledTextSize(scoreText, scoreScale)
		scoreX := (windowW - scoreW) / 2
		window.DrawScaledText(scoreText, scoreX, 0, scoreScale, draw.Black)

		textBackgroundColor := backgroundColor
		textBackgroundColor.A = 0.6
		const textBorderSize = 5

		const highscoreScale = 4
		highscoreText := fmt.Sprintf(" Highscore %d ", highscore)
		highscoreW, highscoreH := window.GetScaledTextSize(highscoreText, highscoreScale)
		highscoreX := windowW - highscoreW
		highscoreYMargin := 10
		highscoreY := highscoreYMargin
		highscoreBottom := highscoreY + highscoreH + highscoreYMargin
		// Fill the text background.
		window.FillRect(highscoreX, 0, 9999, highscoreBottom, textBackgroundColor)
		// Create a fuzzy border around the text background.
		textBorderColor := textBackgroundColor
		for i := range textBorderSize - 1 {
			textBorderColor.A -= textBackgroundColor.A / float32(textBorderSize)
			window.FillRect(highscoreX-1-i, 0, 1, highscoreBottom+i, textBorderColor)
			window.FillRect(highscoreX-1-i, highscoreBottom+i, 9999, 1, textBorderColor)
		}
		// Draw the text on top of the background.
		window.DrawScaledText(highscoreText, highscoreX, highscoreY, highscoreScale, draw.Black)

		// We now draw the gopher name and the kill count in the bottom right
		// hand corner. We want to surround both of these with a single text
		// background rectangle. That is why we do the text size and position
		// calculations first, then draw the background, then draw the texts on
		// top of it.
		const killTextYMargin = 10
		const killScale = 2
		killSuffix := "s"
		if killCount == 1 {
			killSuffix = ""
		}
		killText := fmt.Sprintf(" %d dead gopher%s so far ", killCount, killSuffix)
		killW, killH := window.GetScaledTextSize(killText, killScale)
		killX := windowW - killW
		killY := windowH - killH - killTextYMargin

		// Render the name above the kill count.
		// We blend the text "now playing ..." and "recently deceased ..." when
		// the gopher goes from alive to dead. That is why we render both texts
		// always, but with a different opacity.
		const nameScale = 2
		playingTextAlpha := float32(deceasedTextTime) / deceasedTextFadeFrameCount

		aliveNameText := " now playing: " + name + " "
		aliveNameW, aliveNameH := window.GetScaledTextSize(aliveNameText, nameScale)
		aliveNameX := windowW - aliveNameW
		aliveNameY := killY - aliveNameH
		aliveNameAlpha := (1 - nameAlpha) * playingTextAlpha

		deadNameText := " recently deceased: " + name + " "
		deadNameW, deadNameH := window.GetScaledTextSize(deadNameText, nameScale)
		deadNameX := windowW - deadNameW
		deadNameY := killY - deadNameH
		deadNameAlpha := (1 - nameAlpha) * (1 - playingTextAlpha)

		// Create the text background.
		backX := killX
		if aliveNameAlpha > 0 && aliveNameX < backX {
			backX = aliveNameX
		}
		if deadNameAlpha > 0 && deadNameX < backX {
			backX = deadNameX
		}
		backY := killY - killTextYMargin
		if aliveNameAlpha > 0 || deadNameAlpha > 0 {
			backY = deadNameY - killTextYMargin
		}
		// Fill the text background.
		window.FillRect(backX, backY, 9999, 9999, textBackgroundColor)
		// Create a fuzzy border around the text background.
		textBorderColor = textBackgroundColor
		for i := range textBorderSize - 1 {
			textBorderColor.A -= textBackgroundColor.A / float32(textBorderSize)
			window.FillRect(backX-1-i, backY-i, 1, 9999, textBorderColor)
			window.FillRect(backX-1-i, backY-1-i, 9999, 1, textBorderColor)
		}

		// Write the texts on top of the background.
		window.DrawScaledText(aliveNameText, aliveNameX, aliveNameY, nameScale, draw.RGBA(0, 0, 0, aliveNameAlpha))
		window.DrawScaledText(deadNameText, deadNameX, deadNameY, nameScale, draw.RGBA(0, 0, 0, deadNameAlpha))
		window.DrawScaledText(killText, killX, killY, killScale, draw.RGB(0.7, 0, 0))

		if restartable {
			const text = "Click to Restart"
			textScale := 5 + float32(math.Sin(float64(restartableTime)*0.1))
			restartableTime++
			textW, textH := window.GetScaledTextSize(text, textScale)
			textX := (windowW - textW) / 2
			textY := (windowH - textH) / 2
			window.DrawScaledText(text, textX, textY, textScale, draw.Black)
		}
	})
}

type gap struct {
	centerX           int
	centerY           int
	topPipeShaking    bool
	bottomPipeShaking bool
	shakeTimer        int
}

type circle struct {
	centerX int
	centerY int
	radius  int
}

type rectangle struct {
	left   int
	top    int
	right  int
	bottom int
}

type backgroundTile struct {
	image   string
	x       float64
	yOffset int
}

type cloud struct {
	scale float64
	x     float64
	y     int
}

type kill struct {
	Name        string
	Score       int
	Accessories []string
}

func killsToBytes(kills []kill) []byte {
	var buf bytes.Buffer
	for _, k := range kills {
		buf.WriteString(k.Name)
		buf.WriteString(" ")
		buf.WriteString(strconv.Itoa(k.Score))
		for _, a := range k.Accessories {
			buf.WriteString(" ")
			buf.WriteString(a)
		}
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func bytesToKills(data []byte) []kill {
	var kills []kill
	for line := range strings.SplitSeq(string(data), "\n") {
		cols := strings.Split(line, " ")
		if len(cols) >= 2 {
			name := cols[0]
			score, _ := strconv.Atoi(cols[1])
			accessories := cols[2:]
			kills = append(kills, kill{
				Name:        name,
				Score:       score,
				Accessories: accessories,
			})
		}
	}
	return kills
}

func collides(c circle, r rectangle) bool {
	closestX := min(r.right, max(r.left, c.centerX))
	closestY := min(r.bottom, max(r.top, c.centerY))
	dx := closestX - c.centerX
	dy := closestY - c.centerY
	squareDist := dx*dx + dy*dy
	return squareDist <= c.radius*c.radius
}

func rgb(r, g, b int) draw.Color {
	return rgba(r, g, b, 255)
}

func rgba(r, g, b, a int) draw.Color {
	return draw.RGBA(
		float32(r)/255,
		float32(g)/255,
		float32(b)/255,
		float32(a)/255,
	)
}

func seconds(s float64) time.Duration {
	return time.Duration(round(s * float64(time.Second)))
}

func round(x float64) int {
	if x < 0 {
		return int(x - 0.5)
	}
	return int(x + 0.5)
}
