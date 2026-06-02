; game of life for chip16
; state  = 80x60 cells with 2 cell margin on each side, each cell is 4x4 pixels (size = 5.1kb)
; sprite = 320x40 pixels => 160x40 bytes => 80x10 cells (size = 6.25kb)

DELAY       equ 6           ; max 10 frames/sec
CELL_COLOR  equ #AAAA       ; green cells
STATE_W 	equ 40        	; width in words
STATE_H 	equ 60          ; height in rows
STATE_ROW   equ 84          ; offset to start of next row
SPRITE_BASE	equ #A000		; sprite start
SPRITE_H    equ 40          ; height in pixels
SPRITE_ROWS equ 10          ; height in cells

	spr  #28A0 				; set sprite size
	; -- init initial random state --
	ldi  ra, 0              ; ra = index to source state
	call stateBufferAddr
	mov  rb, r0  			; rb = source address in state
	call randomScreen       ; init new random state

:drawScreen
	; -- draw state from db to screen
	ldi  r0, DELAY
:delayLoop
	cmpi r0, 0
	jz   draw
	vblnk
	subi r0, 1
	jmp  delayLoop
:draw
	cls                     ; draw current state to screen
	ldi  rc, 0              ; rc = sprite x position
	ldi  rd, 0              ; rd = sprite y position
:nextTile
	call copyToSprite
	drw  rc, rd, SPRITE_BASE
	addi rd, SPRITE_H
	cmpi rd, 240
	jl   nextTile

	; -- calc next generation --
	call stateBufferAddr
	mov  rb, r0  			; rb = source buffer pointer
	ldi  r0, 2              
	sub  r0, ra, ra         ; ra = 2 - ra (flip buffers)
	call stateBufferAddr
	mov  rc, r0  			; rc = destination buffer pointer
	ldi  rd, STATE_H        ; rd = row counter
:nextRow
	ldi  re, STATE_W        ; re = column counter in words
:nextCells
	call getCellState 		; r9 = cell state (low byte)
	mov  rf, r9
	addi rb, 1
	call getCellState 		; r9 = cell state (high byte)
	shl  r9, 8
	or   rf, r9
	addi rb, 1
	stm	 rf, rc				; dst cell pair state = rf
	addi rc, 2
	subi re, 1
	jnz  nextCells
	addi rb, 4 				; skip margin
	addi rc, 4
	subi rd, 1
	jnz  nextRow

	call stateBufferAddr     ; rb = start of new buffer
	mov  rb, r0
	jmp  drawScreen


; get new state for cell at rb
; returns r9 = 1 (alive) or 0 (dead), clobbers r0..r2
:getCellState
	ldi  r0, 0         		; r0 = number of neighbours
	mov  r1, rb             
	subi r1, 85             ; top left
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 1              ; top center
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 1              ; top right
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 82             ; left
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 2              ; right
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 82             ; bottom left
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 1              ; bottom center
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	addi r1, 1              ; bottom right
	ldm  r2, r1
	andi r2, #ff
	add  r0, r2
	ldm  r1, rb 			; r1 = prior state
	andi r1, #ff
	ldi  r9, 0
	jz   s_dead
	cmpi r0, 2				; keep existing cell if 2 or 3 neighbours
	jl   s_end
	cmpi r0, 3
	jg   s_end
	ldi  r9, 1
	ret
:s_dead                     ; spawn new cell if 3 neighbours
	cmpi  r0, 3
	jnz   s_end
	ldi   r9, 1
:s_end
	ret


; get start address of buffer with index ra (0 or 2) 
; returns r0, clobbers r1
:stateBufferAddr
	ldi  r1, stateBase
	add  r1, ra
	ldm  r0, r1
	ret

:stateBase
	dw   #C002				; first buffer start
	dw   #E002              ; second buffer start


; copy 10 lines of data from the state buffer starting at rb to the sprite framebuffer
; clobbers r0..r6, updates rb
:copyToSprite
	ldi  r0, SPRITE_BASE    ; r0 = destination address in sprite
	ldi  r1, SPRITE_ROWS 	; r1 = state buffer row counter
:c_copyRow
	ldi  r2, 4              ; r2 = pixels per source row
:c_copyPixelRow
	mov  r3, rb             ; r3 = source address in current row
	ldi  r4, STATE_W        ; r4 = width of each source row in words
:c_nextWord
	ldm  r5, r3             ; r5 = cell state lo + hi -> 2 cells
	addi r3, 2
	ldi  r6, 0              ; draw 4 pixels based on lo(r5)
	tsti r5, #01
	jz   c_loBlank
	ldi  r6, CELL_COLOR
:c_loBlank
	stm  r6, r0
	addi r0, 2
	ldi  r6, 0              ; draw 4 pixels based on hi(r5)
	tsti r5, #0100
	jz   c_hiBlank
	ldi  r6, CELL_COLOR
:c_hiBlank
	stm  r6, r0
	addi r0, 2
	subi r4, 1
	jnz  c_nextWord
	subi r2, 1           	; end of cell row - repeat?
	jnz  c_copyPixelRow
	addi rb, STATE_ROW 		; advance to next row in source
	subi r1, 1
	jnz  c_copyRow
	ret


; initialise random state with 25% chance of each cell being populated
; writes to buffer starting at rb, clobbers r0..r4
:randomScreen
	mov  r0, rb 			; r0 = destination address
	ldi  r1, STATE_H    	; r1 = row counter
:r_nextRow
	ldi  r2, STATE_W     	; r2 = column counter
:r_next
	call randomByte         ; set low byte
	mov  r3, r4 			
	call randomByte         ; set high byte
	shl  r4, 8
	or   r3, r4
	stm  r3, r0             ; store word
	addi r0, 2	
	subi r2, 1   			; next column
	jnz  r_next
	addi r0, 4              ; allow for margins				
	subi r1, 1 				; next row
	jnz  r_nextRow
	ret

; set r4 to 1 if rnd returns 0, 0 if 1,2 or 3
:randomByte
	rnd  r4, 3
	tsti r4, #ff
	jnz  rb_set
	ldi  r4, 1
	ret
:rb_set
	ldi  r4, 0
	ret



