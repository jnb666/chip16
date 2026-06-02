; test drawing some colored rectangles
;
	cls
	bgc 1 					; black background
	spr #0804               ; 8x8 px sprite
	ldi r2, 80              ; number of boxes
:next_box
	rnd r0, 310 			; xpos
	rnd r1, 230             ; ypos
	rnd r3, 1               ; red or green
	cmpi r3, 0
	jz is_red
:is_green
	drw r0, r1, green_box   ; draw at random posn
	jmp continue
:is_red
	drw r0, r1, red_box     ; draw at random posn
:continue
	subi r2, 1
	jnz next_box
:wait
	vblnk 					; wait forever
	jmp wait

:red_box
	db 33h, 33h, 33h, 33h
	db 33h, 33h, 33h, 33h
	db 33h, 00h, 00h, 33h
	db 33h, 00h, 00h, 33h
	db 33h, 00h, 00h, 33h
	db 33h, 00h, 00h, 33h
	db 33h, 33h, 33h, 33h
	db 33h, 33h, 33h, 33h

:green_box
	db 99h, 99h, 99h, 99h
	db 99h, 99h, 99h, 99h
	db 99h, 00h, 00h, 99h
	db 99h, 00h, 00h, 99h
	db 99h, 00h, 00h, 99h
	db 99h, 00h, 00h, 99h
	db 99h, 99h, 99h, 99h
	db 99h, 99h, 99h, 99h