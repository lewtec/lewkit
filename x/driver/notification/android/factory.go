package android

type factory struct{}

func (factory) ID() string   { return "notification_android" }
func (factory) Name() string { return "Android" }
func (factory) Weight() int  { return 80 }
