module main

go 1.25.0

replace gioui.org => ../../.src/gio

replace github.com/gioui-plugins/gio-plugins => ../../.src/gio-plugins

replace github.com/joeblew999/goup-util/pkg/logging => ../../pkg/logging

require (
	gioui.org v0.9.1-0.20251215212054-7bcb315ee174
	github.com/gioui-plugins/gio-plugins v0.9.1
	github.com/joeblew999/goup-util/pkg/logging v0.0.0-00010101000000-000000000000
	golang.org/x/exp/shiny v0.0.0-20250620022241-b7579e27df2b
)

require (
	gioui.org/shader v1.0.8 // indirect
	git.wow.st/gmp/jni v0.0.0-20210610011705-34026c7e22d0 // indirect
	github.com/go-text/typesetting v0.3.0 // indirect
	github.com/inkeliz/gioismobile v0.0.0-20250605191856-aaa9fbad77bc // indirect
	github.com/inkeliz/go_inkwasm v0.1.23-0.20240519174017-989fbe5b10f6 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/image v0.26.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
