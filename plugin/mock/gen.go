package mock

//go:generate mockgen -destination=controller.go -package=$GOPACKAGE github.com/rueian/godemand/plugin Controller
