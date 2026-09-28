package android

type factory struct{}

func (factory) ID() string   { return "confirmer_android" }
func (factory) Name() string { return "Android confirm" }
func (factory) Weight() int  { return 80 }
