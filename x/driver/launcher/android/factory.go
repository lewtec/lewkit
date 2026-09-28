package android

type factory struct{}

func (factory) ID() string   { return "prompter_android" }
func (factory) Name() string { return "Android prompt" }
func (factory) Weight() int  { return 80 }
