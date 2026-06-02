; Note frequencies
C1 	equ 32
C3	equ 131
G3	equ 195
A4	equ 220
B4	equ 247
C4	equ 261
D4	equ 293
E4	equ 329
F4	equ	349
G4	equ 391
A5	equ	440
B5	equ 493
C5	equ 523
D5	equ 587
E5	equ	659
F5	equ 698
G5	equ 783
D6	equ 1174
E6	equ 1318

; Note lengths
d_xs		equ	150
d_vs	 	equ 190
d_m			equ 240
d_n			equ 320
; Wait lengths
w_xs		equ	176
w_vs	 	equ 208
w_m			equ 256
w_n			equ 336
w_l			equ 600
w_loop		equ 1000

init:
	sng 0x64, 0xf196
	ldi r0, 0
	ldi r2, note
	ldi r3, dur
; Traverse a series of notes and play them
play_note:
	mov r4, r0
	addi r4, notes_sonic_ghz
	ldm r1, r4
	cmpi r1, 0
	jz init
	stm r1, r2
	mov r4, r0
	addi r4, dur_sonic_ghz
	ldm r1, r4
	stm r1, r3
	call play
	mov r4, r0
	addi r4, wait_sonic_ghz
	ldm ra, r4
	call wait
	addi r0, 2
	jmp play_note
	
; wait -- Pause the CPU for given number of ms
; ra: number of milliseconds
wait:
	divi ra, 16				; convert from ms to frames
wait_loop:
	cmpi ra, 0
	jz wait_end
	vblnk
	subi ra, 1
	jmp wait_loop
wait_end:
	ret
end:
	vblnk
	jmp end

; Temp note buffer
note:
	dw 0
play:
	db 0x0d, 0x02
dur:
	dw 0
	ret

; Sonic Green Hill Zone melody
; CACBCBGACA AEDCBCBGACE CACBCBGACA AAFAFAFC
notes_sonic_ghz:
	dw C5,A5, C5,B5, C5,B5, G4,A5,C5,A5 
	dw A5,E6,D6, C5,B5, C5,B5, G4,A5,C5,E6
	dw C5,A5, C5,B5, C5,B5, G4,A5,C5,A5
	dw A5,A5,F4, A5,G4, A5,G4,C4
	dw 0
	
dur_sonic_ghz:
	dw d_vs,d_vs, d_vs,d_vs, d_vs,d_vs, d_xs,d_xs,d_xs,d_vs
	dw d_vs,d_vs,d_vs, d_vs,d_vs, d_vs,d_vs, d_xs,d_xs,d_xs,d_xs
	dw d_vs,d_vs, d_vs,d_vs, d_vs,d_vs, d_xs,d_xs,d_xs,d_vs
	dw d_vs,d_vs,d_vs, d_vs,d_vs, d_vs,d_vs,d_vs

wait_sonic_ghz:
	dw w_vs,w_n, w_vs,w_n, w_vs,w_m, w_xs,w_xs,w_vs,w_n
	dw w_vs,w_vs,w_n, w_vs,w_n, w_vs,w_n, w_xs,w_xs,w_vs,w_l
	dw w_vs,w_n, w_vs,w_m, w_vs,w_m, w_xs,w_xs,w_vs,w_n
	dw w_vs,w_vs,w_n, w_vs,w_n, w_vs,w_vs,w_loop