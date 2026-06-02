; test for timing based on VBLNK interval - should wait for 250 msec
;
	ldi r0, 15
:delay
	vblnk			; wait for next frame
	subi r0, 1
	jnz delay
	halt
