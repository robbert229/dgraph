package rdf

import (
	"errors"
	"fmt"
	"io"
)

type parser struct {
	l   *lexer
	err error
}

func newParser(r io.Reader) *parser {
	return &parser{
		l: newLexer(r),
	}
}

func (p *parser) Read() (NQuad, error) {
	return p.nQuad()
}

func (p *parser) nQuad() (ret NQuad, err error) {
	ret.Subject, err = p.subject()
	if err != nil {
		return
	}
	ret.Predicate, err = p.predicate()
	if err != nil {
		return
	}
	err = p.object(&ret)
	if err != nil {
		return
	}
	ret.Label, err = p.label()
	if err != nil {
		return
	}
	err = p.period()
	return
}

func (p *parser) iriRefOrBNLabel() (s string, err error) {
	err = p.l.Err()
	if err != nil {
		return
	}
	t := p.l.Token()
	switch t.Type {
	case iriRef, bnLabel:
		s = t.Value
		p.l.Next()
	default:
		err = fmt.Errorf("unexpected token type %v", t.Type)
	}
	return
}

func (p *parser) subject() (s string, err error) {
	return p.iriRefOrBNLabel()
}

func gotTokenError(issue string, t Token) error {
	return fmt.Errorf("%s, got %s at %s", issue, t.Type, t.Origin)
}

func (p *parser) predicate() (s string, err error) {
	err = p.l.Err()
	if err != nil {
		return
	}
	t := p.l.Token()
	if t.Type != iriRef {
		err = gotTokenError("expected IRIREF", t)
		return
	}
	s = t.Value
	p.l.Next()
	return
}

func (p *parser) object(nquad *NQuad) error {
	if err := p.l.Err(); err != nil {
		return err
	}
	t := p.l.Token()
	switch t.Type {
	case iriRef, bnLabel:
		nquad.ObjectId = t.Value
		p.l.Next()
		return nil
	case literal:
		nquad.ObjectValue = []byte(t.Value)
	default:
		return fmt.Errorf("unexpected token type: %v", t.Type)
	}
	if !p.l.Next() {
		return nil
	}
	t = p.l.Token()
	switch t.Type {
	case doubleHat:
		if !p.l.Next() || p.l.Token().Type != iriRef {
			return errors.New("expected IRIREF after ^^")
		}
		nquad.ObjectValue = append(nquad.ObjectValue, append([]byte("@@"), []byte(p.l.Token().Value)...)...)
		p.l.Next()
	case langTag:
		nquad.Predicate += "." + t.Value
		p.l.Next()
	default:
	}
	return nil
}

func (p *parser) label() (s string, err error) {
	err = p.l.Err()
	if err != nil {
		return
	}
	t := p.l.Token()
	switch t.Type {
	case iriRef, bnLabel:
		s = t.Value
		p.l.Next()
	}
	return
}

func (p *parser) period() (err error) {
	if p.l.Token().Type != period {
		err = errors.New("expected period")
		return
	}
	p.l.Next()
	return
}
