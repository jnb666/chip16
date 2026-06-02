; test program to cycle background colors
;
	cls
	ldi r0, 1 		; r0 = color index
	ldi r1, loop
	addi r1, 2 		; r1 = address of bgc op

:loop
	bgc 1 			; set color
	ldi r2, 60
:wait
	vblnk 			; wait 60 frames
	subi r2, 1
	jnz wait

	addi r0, 1  	; r0++
	cmpi r0, 16             
	jnz update
	ldi r0, 1		; r0 = 1 if r0 == 16
:update
	stm r0, r1		; update bgc op
	jmp loop