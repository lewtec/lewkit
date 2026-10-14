OUTPUT_FORMAT(binary)
ENTRY(_start)
SECTIONS
{
  . = 0x7C00;
  .mbr : { *(.mbr) }
  . = 0x7E00;
  .text : { *(.text) *(.rodata) *(.data) }
}
