package main

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
import "C"

import (
    "unsafe"
    "fmt"
    "bufio"
    "os"
    "log"
    "github.com/gotk3/gotk3/gtk"
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

func inputBoxWidget(input string, session *jamyxgo.Session) *gtk.Widget {
    grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal("Unable to create grid:", err)
	}

    name_label, err := gtk.LabelNew(input)
    if err != nil {
        log.Fatal("Unable to create label:", err)
    }
    vol_slider, err := gtk.ScaleNewWithRange(gtk.ORIENTATION_VERTICAL, 0, 100, 1)
    if err != nil {
        log.Fatal("Unable to create scale:", err)
    }

    vol_slider.Widget.SetVExpand(true)
    vol_slider.SetValue(session.VolumeInputGet(input))
    C.gtk_range_set_inverted((*C.struct__GtkRange)(unsafe.Pointer(vol_slider.Range.GObject)), C.gboolean(1))
    vol_slider.Connect("value_changed", func(){
        session.VolumeInputSet(input, vol_slider.GetValue())
    })

    grid.Widget.SetVExpand(true)
    grid.Attach(name_label, 0, 0, 1, 1)
    grid.Attach(vol_slider, 0, 1, 1, 1)

    go func() {
        local_session := jamyxgo.Session{}
        local_session.Connect("127.0.0.1", 2909)
        for {
            // This is a blocking call waiting for a change in volume and returning it
            vol_slider.SetValue(local_session.VolumeInputListen(input))
        }
    }()

    return &grid.Widget
}

func windowWidget(session *jamyxgo.Session) *gtk.Widget {
    grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal("Unable to create grid:", err)
	}
    grid.Widget.SetVExpand(true)
    grid.Widget.SetHExpand(true)
    // grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

    inputs := session.GetInputs()
    log.Println("Inputs:", inputs)

    var inputBoxes []*gtk.Widget
    for i, in := range inputs {
        inputBoxes = append(inputBoxes, inputBoxWidget(in, session))
        grid.Attach(inputBoxes[i], i, 0, 1, 1)
        // time.Sleep(150 * time.Millisecond)
    }
    for i, inb := range inputBoxes {
        log.Println(i, inb)
    }
    // l, err := gtk.LabelNew("Hello, world!")
    // if err != nil {
    //     log.Fatal("Unable to create label:", err)
    // }
    // grid.Attach(l, 0, 0, 2, 1)
    return &grid.Container.Widget
}

func setupWindow(session *jamyxgo.Session) {
    gtk.Init(nil)
    win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
    if err != nil {
        log.Fatal("Unable to create window:", err)
    }
    win.SetTitle("Simple Example")
    win.Connect("destroy", func() {
        gtk.MainQuit()
    })

    win.Add(windowWidget(session))

    win.SetDefaultSize(500, 300)

    win.ShowAll()
}

func main() {
    session := jamyxgo.Session{}
    session.Connect("127.0.0.1", 2909)

    go interactiveLoop(&session)

    setupWindow(&session)

    gtk.Main()
}
