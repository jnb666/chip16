; simple test program
;
	LDI R0, 3
	LDI R1, 4
	ADD R0, R1		;R0 = 7
	LDI R1, 6		;R1 = 6
	MUL R0, R1, R2	;R2 = 42
	HALT
