## Why was Chip16 started?
Writing an emulator is a famously difficult task. The intricacies of even old processors and systems can make the implementation of an emulator very challenging if one does not know where to start.

Classically, newcomers have been referred to simple systems like Chip8, sometimes the arcade version of Space Invaders, and even the Game Boy, to sink their teeth in.

Chip8 is the best suited of the above for the task: it is simple, relatively well-defined, and there are uncountable emulators for it in existence. However, it was noticed that inconsistencies in the instruction set, several extensions and awkward controls make it more complicated than necessary, particularly for programmers new to emulation.

## How Chip16 solves these problems
Thus, Chip16 was started as a community project, in order to provide a more modern, simple, consistent, yet capable system to emulate.

For full specifications, refer to the [Machine Specification](https://github.com/tykel/chip16/wiki/Machine-Specification) wiki. In short, it is a 1 Mhz RISC-like processor, with 64 KB of RAM, sprite-based color graphics, 8-button controller input, and ADSR waveform capabilities for audio. This allows for more rewarding programs to be developed and emulated.

Unlike Chip8, there is one official specification covering Chip16 -- no extensions or unknown opcodes. An emulator targeting a version of the specification will automatically be compatible with all programs using that version of the specification.

The graphics and input allow Chip16 to more closely resemble video game consoles, which arguably is more fun than a monochrome desktop computer!

## Brief history
In late 2010,  the initial specification versions were finalized, and the first emulators were released targeting them. Over the next year and a half, it has continued to expand with emulators, programs, and interest steadily increasing over time.

I took ownership of the project in November 2011 from ShendoXT, a developer in the pSX emulator community, who started the project.

## Where is Chip16 heading?
That is a good question.

Our first priority is to maintain full backwards compatibility with every specification version. Chip16 has amassed a non-negligible number of programs and emulators, and it would be frustrating for everybody if changes broke everything at every new release.

Our secondary objective is to incrementally add features to Chip16, which should stay in spirit with the project's main objective, simplicity. Proposed changes are added after review, with a new specification version.

As to what is currently being reviewed for addition, we have Modulo, Remainder, Not and Neg operations.
Check the [Issues](https://github.com/tykel/chip16/issues) for feature requests and propositions.