// Destination file declarations. Hosts mount #Tree with Mount.

#SlotText: close({
	kind: "text"
	text: string
})
#SlotRef: close({
	kind: "ref"
	ref:  string
})
#SlotLink: close({
	kind: "link"
	link: string
})
#Slot: string | #SlotText | #SlotRef | #SlotLink

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
#FileLink: close({
	type:   "link"
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
#File: #FileLines | #FileText | #FileRef | #FileLink | #FileStructured

// Tree is a destination map. Keys are fs.FS names.
#Tree: [string]: #File
