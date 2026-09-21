// Destination file declarations. Hosts mount #Tree with Mount.

#SlotText: close({
	kind: "text"
	text: string
})
#SlotRef: close({
	kind: "ref"
	ref:  string
})
#Slot: string | #SlotText | #SlotRef

#FileLines: close({
	type:   "lines"
	mode?:  int
	values: [string]: #Slot
})
#FileText: close({
	type:   "text"
	mode?:  int
	values: [string]: #Slot
})
#FileRef: close({
	type:   "ref"
	mode?:  int
	values: [string]: #Slot
})
// StructuredType is tightened by Mount to the names in the format registry.
#StructuredType: string

#FileStructured: close({
	type:   #StructuredType
	mode?:  int
	values: [string]: _
})
#File: #FileLines | #FileText | #FileRef | #FileStructured

// Tree is a destination map. Keys are fs.FS names.
#Tree: [string]: #File
