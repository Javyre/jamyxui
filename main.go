package main

// // #cgo pkg-config: gtk+-3.0
// // #include <gtk/gtk.h>
// import "C"

// // #include "gtk.go.h"
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

func channelWidget(isinput bool, chan_name string, session *jamyxgo.Session) (channel gtk.IWidget, meter *gtk.ProgressBar) {
    name_label := gtk.NewLabel(chan_name)
    name_label.SetSizeRequest(0, -1)

    initial_vol:= session.VolumeGet(isinput, chan_name)
    var precision int
    if precision=2; initial_vol == 100 { precision = 1 }
    vol_label_text := fmt.Sprintf("%5.[2]*[1]f", initial_vol, precision)

    vol_label  := gtk.NewLabel(vol_label_text)
    vol_slider := gtk.NewVScaleWithRange(0, 100, 1)
    vol_monitor:= gtk.NewProgressBar()
    vol_frame  := gtk.NewFrame("")
    vol_vbox   := gtk.NewVBox(false, 0)
    vol_hbox   := gtk.NewHBox(true, 0)

    is_local_change := false

    vol_label.SetPadding(3, 3)
    vol_slider.SetDrawValue(false)
    vol_slider.SetValue(initial_vol)
    vol_slider.SetInverted(true)
    vol_slider.Connect("value_changed", func(){
        is_local_change = true
        vol := vol_slider.GetValue()
        session.VolumeSet(isinput, chan_name, vol)
        var precision int
        if precision=2; vol == 100 { precision = 1 }
        vol_label.SetText(fmt.Sprintf("%5.[2]*[1]f", vol, precision))
    })
    vol_monitor.SetOrientation(gtk.PROGRESS_BOTTOM_TO_TOP)

    vol_hbox.PackStart(vol_slider , true, true, 0)
    vol_hbox.PackStart(vol_monitor, false, false, 0)
    vol_vbox.PackStart(vol_label, false, false, 0)
    vol_vbox.PackStart(vol_hbox , true , true , 0)
    vol_frame.Add(vol_vbox)

    vbox := gtk.NewVBox(false, 0)
    vbox.PackStart(name_label, false, false, 0)
    vbox.PackStart(vol_frame,  true,  true,  0)

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
            var precision int
            if precision=2; vol == 100 { precision = 1 }
            vol_label.SetText(fmt.Sprintf("%5.[2]*[1]f", vol, precision))
        }
    }()

    return vbox, vol_monitor
}

type Meter struct {
    PortName string
    Port *jack.Port
    MeterGtk *gtk.ProgressBar
    MeterValue float32
}

var g_meters [](*Meter)

func jackProcess(nframes uint32) int {
    for _, meter := range g_meters {
        frames := meter.Port.GetBuffer(nframes)

        // find peak
        var peak float32 = 0
        for _, frame := range frames {
            if float32(frame) > peak { peak = float32(frame) }
        }
        // meter.MeterValue = peak
        // fmt.Println(meter.MeterValue)

        gdk.ThreadsEnter()
        meter.MeterGtk.SetFraction(float64(peak))
        // fmt.Println((*C.GtkProgressBar)(unsafe.Pointer(gtk.PROGRESS_BAR(meter.MeterGtk))))
        gdk.ThreadsLeave()
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
        chan_w, vol_meter := channelWidget(true, in, session)
        hbox.PackStart(chan_w, false, true, 0)

        pn := fmt.Sprintf("jamyxer:%s Out L", in)
        meters = append(meters, &Meter{
            PortName: pn,
            Port: jclient.GetPortByName(pn),
            MeterGtk: vol_meter,
            MeterValue: 0,
        })
    }

    log.Println("Outputs:", outputs)
    for _, out := range outputs {
        chan_w, vol_meter := channelWidget(false, out, session)
        hbox.PackEnd(chan_w, false, true, 0)

        pn := fmt.Sprintf("jamyxer:%s L", out)
        meters = append(meters, &Meter{
            PortName: pn,
            Port: jclient.GetPortByName(pn),
            MeterGtk: vol_meter,
            MeterValue: 0,
        })
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
