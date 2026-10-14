/* Legacy boot for an ELF64 unikernel.
   File layout: LBA 0 = this MBR, LBA 1..32 = stage2, LBA 33.. = ELF.
   Stage2 is executed at 0x7E00. Linked addresses match that. */

	.section .mbr, "ax"
	.code16
	.globl _start
_start:
	cli
	xorw	%ax, %ax
	movw	%ax, %ds
	movw	%ax, %es
	movw	%ax, %ss
	movw	$0x7c00, %sp
	cld
	sti
	movb	%dl, bootdrv

	movw	$dap_mbr, %si
	movb	$0x42, %ah
	movb	bootdrv, %dl
	int	$0x13
	jc	mbr_fail
	ljmp	$0, $stage2

mbr_fail:
	movb	$'!', %al
	call	ser_mbr
1:	hlt
	jmp	1b

ser_mbr:
	pushw	%dx
	pushw	%ax
	movw	$0x3fd, %dx
1:	inb	%dx, %al
	testb	$0x20, %al
	jz	1b
	popw	%ax
	movw	$0x3f8, %dx
	outb	%al, %dx
	popw	%dx
	ret

bootdrv:
	.byte 0

	.align 4
dap_mbr:
	.byte 16, 0
	.word 32
	.word 0x7e00
	.word 0
	.quad 1

	.org 0x1be
	.byte 0x80, 0x00, 0x00, 0x00
	.byte 0x83, 0x00, 0x00, 0x00
	.long 33
	.long 2048
	.org 0x1fe
	.word 0xaa55

	.section .text, "ax"
	.code16
	.globl stage2
stage2:
	cli
	xorw	%ax, %ax
	movw	%ax, %ds
	movw	%ax, %es
	movw	%ax, %ss
	movl	$0x7000, %esp
	cld
	sti

	/* 115200 8N1 */
	movw	$0x3fb, %dx
	movb	$0x80, %al
	outb	%al, %dx
	movw	$0x3f8, %dx
	movb	$1, %al
	outb	%al, %dx
	movw	$0x3f9, %dx
	xorb	%al, %al
	outb	%al, %dx
	movw	$0x3fb, %dx
	movb	$0x03, %al
	outb	%al, %dx
	movw	$0x3fc, %dx
	movb	$0x03, %al
	outb	%al, %dx

	/* 16 sectors of ELF header to 0x2000. */
	movl	$33, %edx
	movw	$16, %bx
	movw	$0x2000, %di
	xorw	%ax, %ax
	call	read_low
	jc	bad
	cmpl	$0x464c457f, 0x2000
	jne	bad

	movl	0x2018, %eax
	movl	%eax, entry_addr
	movl	0x201c, %eax
	movl	%eax, entry_addr+4

	movl	0x2020, %esi
	addl	$0x2000, %esi
	movw	0x2038, %cx		/* e_phnum */
	movw	0x2036, %bp		/* e_phentsize */

phdr_loop:
	testw	%cx, %cx
	jz	enter_long
	cmpl	$1, (%esi)
	jne	phdr_next
	pushw	%cx
	pushl	%esi
	movl	8(%esi), %edx
	movl	24(%esi), %edi
	testl	%edi, %edi
	jnz	1f
	movl	16(%esi), %edi
1:	movl	32(%esi), %ebx
	movl	40(%esi), %eax
	subl	%ebx, %eax
	movl	%eax, bss_left
	call	load_segment
	popl	%esi
	popw	%cx
phdr_next:
	movzwl	%bp, %eax
	addl	%eax, %esi
	decw	%cx
	jmp	phdr_loop

bad:
	movb	$'E', %al
	call	ser
2:	hlt
	jmp	2b

	/* edx=file off, edi=dest, ebx=filesz, bss_left=memsz-filesz */
load_segment:
	movl	%edi, dest_cur
	movl	%edx, off_cur
	movl	%ebx, left_cur
ls_file:
	cmpl	$0, left_cur
	je	ls_bss
	movl	off_cur, %eax
	andl	$511, %eax
	movl	%eax, sec_off
	movl	off_cur, %edx
	shrl	$9, %edx
	addl	$33, %edx
	movw	$64, %bx
	xorw	%di, %di
	movw	$0x1000, %ax
	call	read_low
	jc	bad
	movl	$32768, %ecx
	subl	sec_off, %ecx
	cmpl	left_cur, %ecx
	jbe	1f
	movl	left_cur, %ecx
1:	movl	%ecx, ncopy
	movl	$0x10000, %esi
	addl	sec_off, %esi
	movl	dest_cur, %edi
	call	pm_copy
	movl	ncopy, %ecx
	addl	%ecx, dest_cur
	addl	%ecx, off_cur
	subl	%ecx, left_cur
	jmp	ls_file
ls_bss:
	movl	bss_left, %ecx
	testl	%ecx, %ecx
	jz	1f
	movl	dest_cur, %edi
	call	pm_zero
1:	ret

enter_long:
	cli
	lgdt	gdt_ptr
	movl	%cr0, %eax
	orb	$1, %al
	movl	%eax, %cr0
	ljmpl	$0x08, $pm32

	.code32
pm32:
	movw	$0x10, %ax
	movw	%ax, %ds
	movw	%ax, %es
	movw	%ax, %ss
	movl	$0x9000, %edi
	movl	$0x3000/4, %ecx
	xorl	%eax, %eax
	rep stosl
	movl	$0xa003, 0x9000
	movl	$0xb003, 0xa000
	movl	$0xb000, %edi
	xorl	%eax, %eax
map2m:
	movl	%eax, %ebx
	orl	$0x83, %ebx
	movl	%ebx, (%edi)
	addl	$0x200000, %eax
	addl	$8, %edi
	cmpl	$0x40000000, %eax
	jb	map2m
	movl	$0x9000, %eax
	movl	%eax, %cr3
	movl	%cr4, %eax
	orl	$(1 << 5), %eax
	movl	%eax, %cr4
	movl	$0xc0000080, %ecx
	rdmsr
	orl	$(1 << 8), %eax
	wrmsr
	movl	%cr0, %eax
	orl	$(1 << 31), %eax
	movl	%eax, %cr0
	ljmpl	$0x18, $lm64

	/* Long mode, encoded by hand so this object stays 32-bit. */
lm64:
	.byte 0x31, 0xc0                         /* xor eax, eax */
	.byte 0x8e, 0xd8                         /* mov ds, ax */
	.byte 0x8e, 0xc0                         /* mov es, ax */
	.byte 0x8e, 0xd0                         /* mov ss, ax */
	.byte 0x48, 0xbc
	.quad 0x80000                            /* mov rsp, 0x80000 */
	.byte 0x0f, 0x20, 0xc0                   /* mov rax, cr0 */
	.byte 0x66, 0x83, 0xe0, 0xfb             /* and ax, ~2 (clear EM) */
	.byte 0x66, 0x83, 0xc8, 0x02             /* or ax, MP */
	.byte 0x0f, 0x22, 0xc0                   /* mov cr0, rax */
	.byte 0x0f, 0x20, 0xe0                   /* mov rax, cr4 */
	.byte 0x66, 0x0d, 0x00, 0x06             /* or ax, OSFXSR|OSXMMEXCPT */
	.byte 0x0f, 0x22, 0xe0                   /* mov cr4, rax */
	.byte 0x48, 0x8b, 0x04, 0x25
	.long entry_addr                         /* mov rax, [entry_addr] */
	.byte 0xff, 0xe0                         /* jmp rax */

	.code16
read_low:
	pushw	%si
	movw	%bx, dap_count
	movw	%di, dap_off
	movw	%ax, dap_seg
	movl	%edx, dap_lba
	movl	$0, dap_lba+4
	movw	$dap, %si
	movb	$0x42, %ah
	movb	bootdrv, %dl
	int	$0x13
	popw	%si
	ret

ser:
	pushw	%dx
	pushw	%ax
	movw	$0x3fd, %dx
1:	inb	%dx, %al
	testb	$0x20, %al
	jz	1b
	popw	%ax
	movw	$0x3f8, %dx
	outb	%al, %dx
	popw	%dx
	ret

pm_copy:
	cli
	lgdt	gdt_ptr
	movl	%cr0, %eax
	orb	$1, %al
	movl	%eax, %cr0
	ljmpl	$0x08, $copy32
	.code32
copy32:
	movw	$0x10, %ax
	movw	%ax, %ds
	movw	%ax, %es
	cld
	rep movsb
	jmp	to_real
	.code16
pm_zero:
	cli
	lgdt	gdt_ptr
	movl	%cr0, %eax
	orb	$1, %al
	movl	%eax, %cr0
	ljmpl	$0x08, $zero32
	.code32
zero32:
	movw	$0x10, %ax
	movw	%ax, %ds
	movw	%ax, %es
	cld
	xorb	%al, %al
	rep stosb
to_real:
	ljmpl	$0x20, $pm16
	.code16
pm16:
	movl	%cr0, %eax
	andb	$0xfe, %al
	movl	%eax, %cr0
	ljmp	$0, $real_again
real_again:
	xorw	%ax, %ax
	movw	%ax, %ds
	movw	%ax, %es
	movw	%ax, %ss
	sti
	ret

	.align 8
gdt:
	.quad 0
	.quad 0x00cf9a000000ffff
	.quad 0x00cf92000000ffff
	.quad 0x00af9a000000ffff
	.quad 0x00009a000000ffff	/* 0x20: 16-bit code */
gdt_end:
gdt_ptr:
	.word gdt_end - gdt - 1
	.long gdt

	.align 4
dap:
	.byte 16, 0
dap_count:
	.word 0
dap_off:
	.word 0
dap_seg:
	.word 0
dap_lba:
	.quad 0

entry_addr:
	.quad 0
dest_cur:
	.long 0
off_cur:
	.long 0
left_cur:
	.long 0
sec_off:
	.long 0
ncopy:
	.long 0
bss_left:
	.long 0
