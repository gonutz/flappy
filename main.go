package main

// TODO Sounds:
// - background music

import (
	"embed"
	"fmt"
	"io"
	"math"
	"math/rand"
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
	backgroundImages := []string{
		"rsc/city0.png",
		"rsc/city1.png",
	}
	backgroundColor := rgb(151, 255, 255)
	const (
		windowW, windowH          = 1500, 800
		deadFrame                 = "rsc/dead.png"
		bumpFrame                 = "rsc/bump.png"
		pipeImage                 = "rsc/pipe.png"
		cloudImage                = "rsc/cloud.png"
		gravity                   = 0.5
		gapHeight                 = 300
		gapDistX                  = 600
		gopherX                   = 100
		gopherCollisionRadius     = 50
		clickYSpeed               = -14.0
		minVisiblePipeHeight      = 80
		pipeShakeFrameCount       = 40
		musicIntroFile            = "rsc/music_intro.wav"
		musicIntroLengthInSeconds = 6
		musicLoopFile             = "rsc/music_loop.wav"
		musicLoopLengthInSeconds  = 14
	)

	var (
		animationIndex      int
		nextFlapIn          int
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

	restart := func() {
		animationIndex = 0
		nextFlapIn = 0
		x = 0.0
		y = 400.0
		xSpeed = 5.0
		ySpeed = clickYSpeed
		rotation = 0.0
		targetRotation = 0.0
		isAlive = true
		nextGapX = 1000
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
		highscore = loadHighscore()
		needToPlayFlapSound = true
		playDeathSoundIn = 0
		bumpOnHead = false
		for i := range clouds {
			clouds[i].scale = randomCloudScale()
			clouds[i].x = float64(-350 + rand.Intn(windowW+350))
			clouds[i].y = randomCloudY()
		}
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

		for _, img := range animationFrames {
			preload(img)
		}
		for _, img := range backgroundImages {
			preload(img)
		}
		preload(deadFrame)
		preload(bumpFrame)
		preload(pipeImage)
		preload(cloudImage)

		return preloaded
	}

	imagesAreLoaded := false
	var nextMusicStart time.Time

	draw.RunWindow("Flappy Go", windowW, windowH, func(window draw.Window) {
		window.SetIcon("rsc/icon.png") // TODO Check the prototype/draw ports: this must only happen once.

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

		// Update game state.
		clicked := len(window.Clicks()) > 0 ||
			len(window.Characters()) > 0 ||
			window.WasKeyPressed(draw.KeyUp) ||
			window.WasKeyPressed(draw.KeyEnter) ||
			window.WasKeyPressed(draw.KeyNumEnter)

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
					saveHighscore(highscore)
				}

				scoreAnimationTime = 1.0
			}
		}
		y += ySpeed
		ySpeed += gravity

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
				centerX: gopherX + gopherW/2,
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

		if isAlive {
			window.DrawImageFileRotated(animationFrames[animationIndex], gopherX, round(y), round(rotation))
		} else {
			image := deadFrame
			if bumpOnHead {
				image = bumpFrame
			}
			window.DrawImageFileRotated(image, gopherX, round(y), round(rotation))
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

		const highscoreScale = 4
		highscoreText := fmt.Sprintf("Highscore %d ", highscore)
		highscoreW, _ := window.GetScaledTextSize(highscoreText, highscoreScale)
		highscoreX := windowW - highscoreW
		window.DrawScaledText(highscoreText, highscoreX, 0, highscoreScale, draw.Black)

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
