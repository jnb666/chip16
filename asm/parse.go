package asm

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Main routine to read lines from input stream and append code to a.Code.
func (a *Assembler) Assemble(input io.Reader) error {
	log.Debug("-- pass 1 --")
	a.addr = 0
	if err := a.pass1(input); err != nil {
		return err
	}
	if log.GetLevel() >= log.DebugLevel {
		for name, addr := range a.Labels {
			a.addr2Label[addr] = name
		}
	}
	log.Debug("-- pass 2 --")
	return a.pass2()
}

// parse fields, get label addresses and define constants
func (a *Assembler) pass1(input io.Reader) error {
	scanner := bufio.NewScanner(input)
	line := 0
	filename := basename(input)
	for scanner.Scan() {
		if a.err != nil {
			return fmt.Errorf("error at line %s%d - %w", filename, line, a.err)
		}
		line++
		text := strings.TrimSpace(trimComments(scanner.Text()))
		if len(text) == 0 {
			continue
		}
		fields := parse(text)
		log.Tracef("%04X - %v", a.addr, fields)
		// label definition?
		if label, ok := isLabel(fields[0]); ok {
			a.defineLabel(label, a.addr)
			fields = fields[1:]
		}
		if len(fields) == 0 {
			continue
		}
		// constant definition?
		if len(fields) == 3 && strings.ToUpper(fields[1]) == "EQU" {
			a.defineConstant(fields[0], fields[2])
			continue
		}
		cmd := command{op: strings.ToUpper(fields[0]), args: fields[1:], line: line}
		switch cmd.op {
		case "INCLUDE":
			a.parseInclude(fields[1:])
			continue
		case "IMPORTBIN":
			a.parseImportbin(fields[1:])
			continue
		case "DB":
			code := a.parseDB(cmd.args)
			cmd.size = len(code)
		case "DW":
			cmd.size = 2 * len(cmd.args)
		default:
			cmd.size = 4
		}
		a.commands = append(a.commands, cmd)
		a.addr += cmd.size
	}
	// importbin adds data to end of file
	for _, m := range a.imports {
		a.defineLabel(m.label, a.addr)
		a.addr += len(m.data)
	}
	return scanner.Err()
}

// assemble to byte code in a.Code
func (a *Assembler) pass2() error {
	for _, cmd := range a.commands {
		addr := len(a.Code)
		if name, ok := a.addr2Label[addr]; ok {
			log.Debugf(":%s", name)
		}
		err := a.Instruction(cmd.op, cmd.args)
		opcode := "..."
		if cmd.size <= 4 {
			opcode = fmt.Sprintf("%02X", a.Code[addr:min(addr+cmd.size, len(a.Code))])
		}
		log.Debugf("  %04X  %-8s  %s %s", addr, opcode, cmd.op, strings.Join(cmd.args, ", "))
		if err != nil {
			return fmt.Errorf("syntax error at line %d - %w", cmd.line, err)
		}
	}
	for _, m := range a.imports {
		a.Code = append(a.Code, m.data...)
	}
	if len(a.Code) >= 65536 {
		return fmt.Errorf("output binary size exceeds 64k limit")
	}
	return nil
}

func (a *Assembler) defineLabel(label string, addr int) {
	if _, ok := a.Labels[label]; ok {
		a.err = fmt.Errorf("duplicate label %q", label)
		return
	}
	log.Debugf("label %s = %04X", label, addr)
	a.Labels[label] = addr
}

func (a *Assembler) defineConstant(name, value string) {
	if _, ok := a.Constants[name]; ok {
		a.err = fmt.Errorf("duplicate constant %q", name)
		return
	}
	log.Debugf("const %s = %s", name, value)
	a.Constants[name] = value
}

func (a *Assembler) parseDB(args []string) (code []byte) {
	if len(args) == 0 {
		a.err = fmt.Errorf("DB: missing operand")
		return nil
	}
	a.op = "DB"
	for _, constant := range args {
		if constant[0] == '"' {
			str, err := strconv.Unquote(constant)
			if err != nil {
				a.err = fmt.Errorf("DB: %s %w", constant, err)
				return nil
			}
			code = append(code, []byte(str)...)
		} else {
			b := a.byte(constant)
			if a.err != nil {
				return
			}
			code = append(code, b)
		}
	}
	return code
}

func (a *Assembler) parseDW(args []string) (code []byte) {
	if len(args) == 0 {
		a.err = fmt.Errorf("DW: missing operand")
		return nil
	}
	a.op = "DW"
	for _, constant := range args {
		w := a.num(constant)
		if a.err != nil {
			return nil
		}
		code = binary.LittleEndian.AppendUint16(code, w)
	}
	return code
}

func (a *Assembler) parseInclude(args []string) {
	if len(args) != 1 {
		a.err = fmt.Errorf("include: expecting 1 arg")
		return
	}
	f, err := os.Open(a.filename(args[0]))
	if err != nil {
		a.err = fmt.Errorf("include: error opening %s - %w", args[0], err)
		return
	}
	defer f.Close()
	a.err = a.pass1(f)
}

// importbin FILE OFFSET LENGTH LABEL
func (a *Assembler) parseImportbin(args []string) {
	if len(args) != 4 {
		a.err = fmt.Errorf("importbin: expecting 4 args")
		return
	}
	label := args[3]
	if !isIdent(label) {
		a.err = fmt.Errorf("importbin: invalid label: %q", label)
		return
	}
	offset, err := atoi(getBase(args[1]))
	if err != nil || offset < 0 {
		a.err = fmt.Errorf("importbin: invalid offset: %w", err)
		return
	}
	length, err := atoi(getBase(args[2]))
	if err != nil || length <= 0 {
		a.err = fmt.Errorf("importbin: invalid length: %w", err)
		return
	}
	data, err := os.ReadFile(a.filename(args[0]))
	if err != nil {
		a.err = fmt.Errorf("importbin: error reading data from %s: %w", args[0], err)
		return
	}
	a.imports = append(a.imports, importbin{label: label, data: data[offset:min(offset+length, len(data))]})
}

func (a *Assembler) filename(file string) string {
	if !filepath.IsAbs(file) && a.BaseDir != "" {
		file = filepath.Join(a.BaseDir, file)
	}
	return file
}

func basename(r io.Reader) string {
	if file, ok := r.(*os.File); ok {
		return filepath.Base(file.Name()) + ":"
	}
	return ""
}

func parse(line string) (fields []string) {
	for {
		var tok string
		tok, line = parseToken(line)
		if tok == "" {
			return
		}
		fields = append(fields, tok)
	}
}

func parseToken(line string) (token, rest string) {
	inString := false
	runes := []rune(line)
	for i, ch := range runes {
		if inString {
			if ch == '"' && runes[i-1] != '\\' {
				token += `"`
				return
			}
			token += string(ch)
		} else {
			switch ch {
			case ' ', '\t', ',':
				if token != "" {
					rest = line[i+1:]
					return
				}
			case '"':
				inString = true
				token += string(ch)
			default:
				token += string(ch)
			}
		}
	}
	return
}

func trimComments(line string) string {
	for i, ch := range []rune(line) {
		if ch == ';' {
			return line[:i]
		}
	}
	return line
}

func isLabel(token string) (string, bool) {
	if len(token) < 2 {
		return "", false
	}
	if token[0] == ':' && isIdent(token[1:]) {
		return token[1:], true
	}
	if n := len(token) - 1; token[n] == ':' && isIdent(token[:n]) {
		return token[:n], true
	}
	return "", false
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if !(ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || i > 0 && ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}
