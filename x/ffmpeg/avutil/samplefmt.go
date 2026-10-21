package avutil

// BytesPerSample is the storage of one sample in this format.
func BytesPerSample(format SampleFormat) int {
	switch format {
	case SampleU8, SampleU8P:
		return 1
	case SampleS16, SampleS16P:
		return 2
	case SampleS32, SampleS32P, SampleFLT, SampleFLTP:
		return 4
	case SampleS64, SampleS64P, SampleDBL, SampleDBLP:
		return 8
	default:
		return 0
	}
}

// IsPlanarSample reports a planar PCM format.
func IsPlanarSample(format SampleFormat) bool {
	switch format {
	case SampleU8P, SampleS16P, SampleS32P, SampleS64P, SampleFLTP, SampleDBLP:
		return true
	default:
		return false
	}
}

// PackedSample is the interleaved twin of format.
func PackedSample(format SampleFormat) SampleFormat {
	switch format {
	case SampleU8P:
		return SampleU8
	case SampleS16P:
		return SampleS16
	case SampleS32P:
		return SampleS32
	case SampleS64P:
		return SampleS64
	case SampleFLTP:
		return SampleFLT
	case SampleDBLP:
		return SampleDBL
	default:
		return format
	}
}

// PlanarSample is the planar twin of format.
func PlanarSample(format SampleFormat) SampleFormat {
	switch PackedSample(format) {
	case SampleU8:
		return SampleU8P
	case SampleS16:
		return SampleS16P
	case SampleS32:
		return SampleS32P
	case SampleS64:
		return SampleS64P
	case SampleFLT:
		return SampleFLTP
	case SampleDBL:
		return SampleDBLP
	default:
		return format
	}
}
