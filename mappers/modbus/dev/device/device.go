package device

func DevStart() {
	for _, dev := range devices {
		Start(dev)
	}
}
