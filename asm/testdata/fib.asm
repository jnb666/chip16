; simple fibonacci calc
;
	ldi r0, 16		;number of elements to generate
	ldi r1, #0100 	;output location
	ldi r2, 1		;first
	ldi r3, 1 		;second
	stm r2, r1
	push r2
:loop
	addi r1, 2
	stm r3, r1
	push r3
	mov r4, r2      ;save first 
	mov r2, r3      ;first -> second
	add r3, r4      ;second -> first+second

	subi r0, 1		;check if done
	jnz loop

	halt
