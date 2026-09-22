package vulkan

import "encoding/binary"

// SmokeSPIRV writes 2 to the uint32 at storage-buffer binding 0.
// Local size is 1×1×1. Use it to check that a compute dispatch lands.
func SmokeSPIRV() []byte {
	const (
		opCapability     = 17
		opMemoryModel    = 14
		opEntryPoint     = 15
		opExecutionMode  = 16
		opDecorate       = 71
		opMemberDecorate = 72
		opTypeVoid       = 19
		opTypeFunction   = 33
		opTypeInt        = 21
		opTypePointer    = 32
		opTypeStruct     = 30
		opVariable       = 59
		opConstant       = 43
		opFunction       = 54
		opFunctionEnd    = 56
		opLabel          = 248
		opAccessChain    = 65
		opStore          = 62
		opReturn         = 253

		idVoid = 1
		idFn   = 2
		idU32  = 3
		idBuf  = 4
		idPtrB = 5
		idData = 6
		idPtrU = 7
		idZero = 8
		idTwo  = 9
		idMain = 10
		idLab  = 11
		idElem = 12

		uniform     = 2
		bufferBlock = 3
	)
	inst := func(op uint32, args ...uint32) []uint32 {
		return append([]uint32{(uint32(1+len(args)) << 16) | op}, args...)
	}
	words := []uint32{
		0x07230203,
		0x00010000,
		0,
		13,
		0,
	}
	words = append(words, inst(opCapability, 1)...)
	words = append(words, inst(opMemoryModel, 0, 1)...)
	words = append(words, (6<<16)|opEntryPoint, 5, idMain, 0x6e69616d, 0, idData)
	words = append(words, inst(opExecutionMode, idMain, 17, 1, 1, 1)...)
	words = append(words, inst(opMemberDecorate, idBuf, 0, 35, 0)...)
	words = append(words, inst(opDecorate, idBuf, bufferBlock)...)
	words = append(words, inst(opDecorate, idData, 34, 0)...)
	words = append(words, inst(opDecorate, idData, 33, 0)...)
	words = append(words, inst(opTypeVoid, idVoid)...)
	words = append(words, inst(opTypeFunction, idFn, idVoid)...)
	words = append(words, inst(opTypeInt, idU32, 32, 0)...)
	words = append(words, inst(opTypeStruct, idBuf, idU32)...)
	words = append(words, inst(opTypePointer, idPtrB, uniform, idBuf)...)
	words = append(words, inst(opVariable, idPtrB, idData, uniform)...)
	words = append(words, inst(opTypePointer, idPtrU, uniform, idU32)...)
	words = append(words, inst(opConstant, idU32, idZero, 0)...)
	words = append(words, inst(opConstant, idU32, idTwo, 2)...)
	words = append(words, inst(opFunction, idVoid, idMain, 0, idFn)...)
	words = append(words, inst(opLabel, idLab)...)
	words = append(words, inst(opAccessChain, idPtrU, idElem, idData, idZero)...)
	words = append(words, inst(opStore, idElem, idTwo)...)
	words = append(words, inst(opReturn)...)
	words = append(words, inst(opFunctionEnd)...)
	out := make([]byte, len(words)*4)
	for i, w := range words {
		binary.LittleEndian.PutUint32(out[i*4:], w)
	}
	return out
}
