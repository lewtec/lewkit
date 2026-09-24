package gui

import "image"

type hitKey struct {
	key    string
	bounds Rect
	clip   Rect
}

func (picture *Picture) noteKey(key string, bounds, clip Rect) {
	if picture == nil || key == "" {
		return
	}
	picture.keys = append(picture.keys, hitKey{key: key, bounds: bounds, clip: clip})
}

// Hit is the topmost keyed box under at. The second and third results are
// the point inside that box, 0 at the start and 1 at the end.
func Hit(root Node, size Size, at image.Point) (string, float32, float32) {
	picture, err := NewPicture()
	if err != nil || root == nil {
		return "", 0, 0
	}
	picture.recordOnly = true
	if _, err := picture.Render(root, size); err != nil {
		return "", 0, 0
	}
	x, y := float32(at.X), float32(at.Y)
	found := hitKey{}
	for _, key := range picture.keys {
		if !inside(x, y, key.clip) || !inside(x, y, key.bounds) {
			continue
		}
		found = key
	}
	if found.key == "" || found.bounds.Width < 1 || found.bounds.Height < 1 {
		return "", 0, 0
	}
	return found.key, (x - found.bounds.X) / found.bounds.Width, (y - found.bounds.Y) / found.bounds.Height
}

func inside(x, y float32, rect Rect) bool {
	return x >= rect.X && x < rect.X+rect.Width && y >= rect.Y && y < rect.Y+rect.Height
}
