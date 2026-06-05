; Example music-playing program for chip16
; Plays a list of sounds sequentially.

; You may reuse this code, credit preferred.
; (C) tykel, 2012

; Note frequencies
C4      equ 262
D4      equ 294
DS4     equ 311
E4      equ 330
F4      equ 349
G4      equ 392
GS4     equ 415
A4      equ 440
B4      equ 494
C5      equ 523
D5      equ 587
E5      equ 659
F5      equ 698
G5      equ 784
A5      equ 880
B5      equ 988

; Note lengths
; 120bpm
d_sxt_120       equ 125         ; 1/16
d_eht_120       equ 250         ; 1/8
d_qtr_120       equ 500         ; 1/4
d_hlf_120       equ 1000        ; 1/2
d_whl_120       equ 2000        ; 1/1, probably not used
d_3sxt_120      equ 375         ; 3/16 = 1/8 .
d_3eht_120      equ 750         ; 3/8 = 1/4 .
d_3qtr_120      equ 1500        ; 3/4 = 1/2 .

init:
    ; sng AD, VTSR
    sng 0x14, 0xf1d7
    ldi r0, 0                   ; offset
    ldi r2, note                ; r2 = &frequency
    ldi r3, dur                 ; r3 = &duration
; Traverse a series of notes and play them
play_note:
    mov r4, r0                  ; r4 = note array ptr
    addi r4, notes_test 
    ldm r1, r4                  ; r1 = note to play
    cmpi r1, 0xFFFF
    jz end
    stm r1, r2                  ; *frequency = note
    mov r5, r1                  ; r5 = note
    mov r4, r0                  ; r4 = dur array ptr
    addi r4, dur_test 
    ldm r1, r4                  ; r1 = duration of note
    stm r1, r3                  ; *duration = d
    call play
play_note_wait:
    mov ra, r1
    call wait
    addi r0, 2
    jmp play_note

; wait -- Pause the CPU for given number of ms
; ra: number of milliseconds
wait:
    divi ra, 16             ; convert from ms to frames
wait_loop:
    cmpi ra, 0
    jz wait_end
    vblnk
    subi ra, 1
    jmp wait_loop
wait_end:
    ret

end:
    vblnk                       ; wait forever
    jmp end

; Temp note buffer
note:
    dw 0
play:
    db 0x0d, 0x02               ; snp r2, dur
dur:
    dw 0                      
    ret

notes_test:
    dw D4, E4, F4, G4, A4, F4, A4, GS4, E4
    dw GS4, G4, DS4, G4, D4, E4, F4, G4, A4, F4, A4
    dw D5, C5, A4, F4, A4, C5
    dw 0xFFFF

dur_test:
    dw d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_qtr_120, d_eht_120, d_eht_120
    dw d_qtr_120, d_eht_120, d_eht_120, d_qtr_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120
    dw d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_eht_120, d_hlf_120 

