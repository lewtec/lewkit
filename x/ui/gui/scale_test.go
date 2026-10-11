package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/basicfont"
)

func TestBoxScalesAuthoredLengths(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.SetScale(2)
	root := &Stack{Children: []Node{
		&Positioned{Child: &Box{Width: 10, Height: 8, Radius: 3, Fill: &RGB{1, 2, 3, 255}}},
	}}
	_, err = picture.Render(root, Size{200, 200})
	require.NoError(t, err)
	require.NotEmpty(t, picture.fills)
	assert.Equal(t, float32(20), picture.fills[0].Width)
	assert.Equal(t, float32(16), picture.fills[0].Height)
	assert.Equal(t, float32(6), picture.fills[0].Radius)
}

func TestTextBitmapScales(t *testing.T) {
	one := &Text{Value: "M", Face: basicfont.Face7x13}
	base := one.Layout(Tight(200, 100))
	two := &Text{Value: "M", Face: basicfont.Face7x13}
	constraints := Tight(400, 200)
	constraints.scale = 2
	got := two.Layout(constraints)
	assert.Equal(t, base.Width*2, got.Width)
	assert.Equal(t, base.Height*2, got.Height)
}

func TestTextFollowsFontPreference(t *testing.T) {
	loose := BoxConstraints{MaxWidth: 400, MaxHeight: 200}
	one := &Text{Value: "M", Face: basicfont.Face7x13}
	base := one.Layout(loose)
	scaled := &Text{Value: "M", Face: basicfont.Face7x13}
	constraints := loose
	constraints.scale = 2
	constraints.font = 1.5
	got := scaled.Layout(constraints)
	assert.Equal(t, base.Width*3, got.Width)
	assert.Equal(t, base.Height*3, got.Height)

	picture, err := NewPicture()
	require.NoError(t, err)
	picture.SetScale(2)
	picture.SetFontScale(1.5)
	_, err = picture.Render(&Stack{Children: []Node{
		&Positioned{Child: &Box{Width: 10, Height: 8, Fill: &RGB{1, 2, 3, 255}}},
	}}, Size{200, 200})
	require.NoError(t, err)
	require.NotEmpty(t, picture.fills)
	assert.Equal(t, float32(20), picture.fills[0].Width)
	assert.Equal(t, float32(16), picture.fills[0].Height)

	caption := &Text{Value: "M", Face: basicfont.Face7x13}
	_, err = picture.Render(&Stage{Child: &Positioned{Child: caption}}, Size{200, 80})
	require.NoError(t, err)
	assert.Equal(t, base.Width*3, caption.size.Width)
	assert.Equal(t, base.Height*3, caption.size.Height)
}

func TestFlexGapScales(t *testing.T) {
	loose := BoxConstraints{MaxWidth: 400, MaxHeight: 400}
	plain := Row(&Box{Width: 10, Height: 8}, &Box{Width: 10, Height: 8})
	plain.Gap = 4
	base := plain.Layout(loose)
	assert.Equal(t, float32(24), base.Width)
	assert.Equal(t, float32(8), base.Height)

	scaled := Row(&Box{Width: 10, Height: 8}, &Box{Width: 10, Height: 8})
	scaled.Gap = 4
	loose.scale = 2
	got := scaled.Layout(loose)
	assert.Equal(t, float32(48), got.Width)
	assert.Equal(t, float32(16), got.Height)
}
