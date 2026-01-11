package main

import (
	"log"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/gordonklaus/portaudio"
)

var (
	stream      *portaudio.Stream
	running     bool
	activityLed *widget.Label
)

func main() {
	_ = portaudio.Initialize()
	defer portaudio.Terminate()

	a := app.NewWithID("mic.keeper")
	prefs := a.Preferences()

	// devices
	devices, _ := portaudio.Devices()
	micMap := map[string]*portaudio.DeviceInfo{}
	micNames := []string{}

	for _, d := range devices {
		if d.MaxInputChannels > 0 {
			micNames = append(micNames, d.Name)
			micMap[d.Name] = d
		}
	}

	w := a.NewWindow("Microphone Keeper")

	status := widget.NewLabel("Idle")
	activityLed = widget.NewLabel("○ Inactive")

	sampleRateEntry := widget.NewEntry()
	selectMic := widget.NewSelect(micNames, func(name string) {
		if d := micMap[name]; d != nil {
			sampleRateEntry.SetText(strconv.Itoa(int(d.DefaultSampleRate)))
		}
	})

	// restore saved settings
	selectMic.SetSelected(prefs.String("mic"))
	sampleRateEntry.SetText(prefs.StringWithFallback("samplerate", ""))

	startBtn := widget.NewButton("Start", func() {
		if running {
			return
		}
		device := micMap[selectMic.Selected]
		if device == nil {
			status.SetText("Select microphone")
			return
		}

		sr, err := strconv.Atoi(sampleRateEntry.Text)
		if err != nil || sr <= 0 {
			sr = int(device.DefaultSampleRate)
		}

		buffer := make([]int16, 64)
		params := portaudio.HighLatencyParameters(device, nil)
		params.Input.Channels = 1
		params.SampleRate = float64(sr)
		params.FramesPerBuffer = len(buffer)

		stream, err = portaudio.OpenStream(params, buffer)
		if err != nil {
			status.SetText("Open error")
			log.Println(err)
			return
		}
		stream.Start()

		prefs.SetString("mic", selectMic.Selected)
		prefs.SetString("samplerate", strconv.Itoa(sr))

		running = true
		status.SetText("Running")
		activityLed.SetText("● Active")

		go func() {
			for running {
				stream.Read()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	})

	stopBtn := widget.NewButton("Stop", func() {
		if stream != nil {
			stream.Stop()
			stream.Close()
		}
		running = false
		status.SetText("Stopped")
		activityLed.SetText("○ Inactive")
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Microphone"),
		selectMic,
		widget.NewLabel("Sample Rate"),
		sampleRateEntry,
		container.NewHBox(startBtn, stopBtn),
		activityLed,
		status,
	))

	// ---- System Tray ----
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("MicKeeper",
			fyne.NewMenuItem("Show", func() { w.Show() }),
			fyne.NewMenuItem("Hide", func() { w.Hide() }),
			fyne.NewMenuItem("Quit", func() {
				if stream != nil {
					stream.Stop()
				}
				a.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
	}
	w.Resize(fyne.NewSize(600, 500))
	w.ShowAndRun()
}
