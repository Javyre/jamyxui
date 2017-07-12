package main

// // #cgo pkg-config: gtk+-3.0
// // #include <gtk/gtk.h>
// import "C"

// #include <gtk/gtk.h>
// #cgo pkg-config: gtk+-2.0
import "C"

import (
    // "unsafe"
    // "time"
    "fmt"
    "bufio"
    "os"
    "log"
    // "math"
    "github.com/xthexder/go-jack"
    "github.com/mattn/go-gtk/gdk"
    "github.com/mattn/go-gtk/gtk"
    "github.com/javyre/jamyxgo"
)


func interactiveLoop(session *jamyxgo.Session) {
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("Command: ")
        cmd, _ := reader.ReadString('\n')
        log.Println(session.SendCommand(cmd))
    }
}

type gtkMeter  struct { *gtk.DrawingArea }
type gdkPixmap struct { *gdk.Pixmap }

func (pm *gdkPixmap) Fill(gc *gdk.GC) {
    pm.GetDrawable().DrawRectangle(gc, true, 0, 0, -1, -1)
}

type Meter struct {
    PortName string
    PortL *jack.Port
    PortR *jack.Port
    MeterGtk *gtkMeter
    MeterValueL *float32
    MeterValueR *float32
}

func gtkNewMeter(isStereo bool, meterValues [](*float32)) *gtkMeter {
    // Macro
    invertCoords := func(t_w, t_h, x, y, w, h int) (rx, ry, rw, rh int) {
        rx, ry, rw, rh = x, y, w, h

        rx = t_w - (x + w)
        ry = t_h - (y + h)

        return rx, ry, w, h
    }
    propCoords := func(t_w, t_h int, x, y, w, h float32) (rx, ry, rw, rh int) {
        t_w_f := float32(t_w)
        t_h_f := float32(t_h)

        rx = int(t_w_f * x)
        ry = int(t_h_f * y)
        rw = int(t_w_f * w)
        rh = int(t_h_f * h)

        return rx, ry, rw, rh
    }

    meter := gtk.NewDrawingArea()

    var pixmap *gdkPixmap
    var bgGC, barGC, _GC *gdk.GC
    var gdkwin *gdk.Window

    var vm_width, vm_height int

    bgColor  := gdk.NewColor("black")
    barColor := gdk.NewColor("blue")

    // Config event
    meter.Connect("configure-event", func(){
        if pixmap != nil { pixmap.Unref() }
        this                := meter
        allocation          := this.GetAllocation()
        vm_width, vm_height  = allocation.Width, allocation.Height

        gdkwin      = this.GetWindow()
        pixmap      = &gdkPixmap{gdk.NewPixmap(gdkwin.GetDrawable(), vm_width, vm_height, 24)}
        bgGC        = gdk.NewGC(pixmap.GetDrawable())
        barGC       = gdk.NewGC(pixmap.GetDrawable())
        _GC         = gdk.NewGC(pixmap.GetDrawable())

        bgGC .SetRgbFgColor(bgColor)
        barGC.SetRgbFgColor(barColor)

        pixmap.Fill(bgGC)
    })

    var curLevelL, curLevelR float32 = 0, 0
    // Expose event
    meter.Connect("expose-event", func() {
		if pixmap == nil {
			return
		}
        // Draw bg
        pixmap.Fill(bgGC)

        curLevelL -= 0.04
        if curLevelL < 0 { curLevelL = 0 }
        if *meterValues[0] > curLevelL {
            curLevelL = *meterValues[0]
        }

        curLevelR -= 0.04
        if curLevelR < 0 { curLevelR = 0 }
        if *meterValues[1] > curLevelR {
            curLevelR = *meterValues[1]
        }

        // Draw bars
        x, y, w, h := propCoords(vm_width, vm_height, 0.5, 0, 0.5, curLevelL)
        x, y, w, h  = invertCoords(vm_width, vm_height, x, y, w, h)
        pixmap.GetDrawable().DrawRectangle(barGC, true, x, y, w, h)

        x, y, w, h  = propCoords(vm_width, vm_height, 0, 0, 0.5, curLevelR)
        x, y, w, h  = invertCoords(vm_width, vm_height, x, y, w, h)
        pixmap.GetDrawable().DrawRectangle(barGC, true, x, y, w, h)

        // Display everything
		gdkwin.GetDrawable().DrawDrawable(_GC, pixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)

        // vol_monitor2.QueueDraw()
	})

    return &gtkMeter{meter}
}

func channelWidget(isinput bool,
                   chan_name string,
                   session *jamyxgo.Session,
                   jclient *jack.Client,
               ) (widget gtk.IWidget, meter *Meter) {

    // Macro
    getPrecision := func(vol float64) int {
        if vol == 100 { return 1 }
        if vol == 0   { return 3 }
        return 2
    }
    getVolLabelText := func(vol float64) string {
        return fmt.Sprintf("%5.[2]*[1]f", vol, getPrecision(vol))
    }

    // ==== Initialize values ====
    var meterValL float32 = 0.5
    var meterValR float32 = 0.5
    meterValues := [](*float32){ &meterValL, &meterValR }

    initial_vol:= session.VolumeGet(isinput, chan_name)

    // ==== Initialize gtk objects ====
    name_label  := gtk.NewLabel(chan_name)
    vol_label   := gtk.NewLabel(getVolLabelText(initial_vol))
    vol_slider  := gtk.NewVScaleWithRange(0, 100, 1)
    vol_monitor := gtkNewMeter(true, meterValues)
    vol_frame   := gtk.NewFrame("")
    vol_vbox    := gtk.NewVBox(false, 0)
    vol_hbox    := gtk.NewHBox(true, 0)
    vbox        := gtk.NewVBox(false, 0)

    is_local_change := false

    // ==== Configure gtk objects ====
    name_label.SetSizeRequest(0, -1)

    vol_label.SetPadding(3, 3)
    vol_slider.SetDrawValue(false)
    vol_slider.SetValue(initial_vol)
    vol_slider.SetInverted(true)

    // ==== Place gtk objects ====
    vol_hbox.PackStart(vol_slider,  true,  true,  0)
    vol_hbox.PackStart(vol_monitor, true,  true,  0)

    vol_vbox.PackStart(vol_label,   false, false, 0)
    vol_vbox.PackStart(vol_hbox,    true,  true,  0)

    vol_frame.Add(vol_vbox)

    vbox.PackStart(name_label, false, false, 0)
    vbox.PackStart(vol_frame,  true,  true,  0)

    // ==== Set callbacks and goroutines ====
    vol_slider.Connect("value_changed", func(){
        is_local_change = true
        vol := vol_slider.GetValue()
        session.VolumeSet(isinput, chan_name, vol)
        vol_label.SetText(getVolLabelText(vol))
    })

    // go func() {
    //     for {
    //         gdk.ThreadsEnter()
    //         vol_monitor.QueueDraw()
    //         gdk.ThreadsLeave()
    //         time.Sleep(16 * time.Millisecond)
    //     }
    // }()

    go func() {
        local_session := jamyxgo.Session{}
        local_session.Connect("127.0.0.1", 2909)
        for {
            // This is a blocking call waiting for a change in volume and returning it
            vol := local_session.VolumeListen(isinput, chan_name)
            if is_local_change {
                is_local_change = false
                continue
            }
            vol_slider.SetValue(vol)
            vol_label.SetText(getVolLabelText(vol))
        }
    }()

    // Construct Meter object
    // return vbox, vol_monitor, &vol_meter_val
    suffix := ""; if isinput { suffix="Out "}

    meter             = new(Meter)
    meter.PortName    = fmt.Sprintf("jamyxer:%s %s", chan_name, suffix)
    meter.PortL       = jclient.GetPortByName(meter.PortName+"L")
    meter.PortR       = jclient.GetPortByName(meter.PortName+"R")
    meter.MeterGtk    = vol_monitor
    meter.MeterValueL = &meterValL
    meter.MeterValueR = &meterValR
    return vbox, meter
}

var g_meters [](*Meter)

func jackProcess(nframes uint32) int {
    for _, meter := range g_meters {
        framesL := meter.PortL.GetBuffer(nframes)
        framesR := meter.PortR.GetBuffer(nframes)

        // find peak
        var peakL float32 = 0
        var peakR float32 = 0
        for _, frame := range framesL {
            if float32(frame) > peakL { peakL = float32(frame) }
        }
        for _, frame := range framesR {
            if float32(frame) > peakR { peakR = float32(frame) }
        }
        *meter.MeterValueL = peakL
        *meter.MeterValueR = peakR

        gdk.ThreadsEnter()
        meter.MeterGtk.QueueDraw()
        gdk.ThreadsLeave()
        // fmt.Println(meter.MeterValue)

        // gdk.ThreadsEnter()
        // meter.MeterGtk.SetFraction(float64(peak))
        // // fmt.Println((*C.GtkProgressBar)(unsafe.Pointer(gtk.PROGRESS_BAR(meter.MeterGtk))))
        // gdk.ThreadsLeave()
    }
    return 0
}

func windowWidget(session *jamyxgo.Session, jclient *jack.Client) gtk.IWidget {
    hbox := gtk.NewHBox(false, 0)

    inputs  := session.GetInputs()
    outputs := session.GetOutputs()

    var meters []*Meter

    log.Println("Inputs:", inputs)
    for _, in := range inputs {
        chan_w, meter := channelWidget(true, in, session, jclient)
        hbox.PackStart(chan_w, false, true, 0)

        meters = append(meters, meter)
    }

    log.Println("Outputs:", outputs)
    for _, out := range outputs {
        chan_w, meter := channelWidget(false, out, session, jclient)
        hbox.PackEnd(chan_w, false, true, 0)

        meters = append(meters, meter)
    }
    g_meters = meters


    return hbox
}

func setupWindow(session *jamyxgo.Session, jclient *jack.Client) {
    gdk.ThreadsInit()
    gtk.Init(nil)
    window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("Jamyxui")
    window.Connect("destroy", gtk.MainQuit)

    window.Add(windowWidget(session, jclient))

    window.SetSizeRequest(-1, 300)
    window.ShowAll()
}

func setupJack(session *jamyxgo.Session) *jack.Client {
    client, _ := jack.ClientOpen("Jamyxui channels monitor", jack.NoStartServer)
    if client == nil { log.Fatal("Could not connect to jack server!") }

    client.SetProcessCallback(jackProcess)

    if code := client.Activate(); code != 0 { log.Fatal("Failed to activate client!") }

    // go func() {
    //     for {
    //         gdk.ThreadsEnter()
    //         for _, meter := range g_meters {
    //             meter.MeterGtk.SetFraction(float64(meter.MeterValue))
    //             // fmt.Println(float64(meter.MeterValue))
    //         }
    //         gdk.ThreadsLeave()
    //         time.Sleep(8 * time.Millisecond)
    //     }
    // }()

    return client
}

func main() {
    session := jamyxgo.Session{}
    session.Connect("127.0.0.1", 2909)

    go interactiveLoop(&session)

    jclient := setupJack(&session)
    defer jclient.Close()

    fmt.Println(jclient.GetPorts("jamyxer:.*", ".*", 0))

    setupWindow(&session, jclient)

    gtk.Main()
}
