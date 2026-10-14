package cliente

import (
	"time"

	"github.com/google/uuid"
)

const tamanhoMaximoNomeGrupo = 120

type Grupo struct {
	ID           uuid.UUID
	Nome         string
	NomeBusca    string
	Ativo        bool
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

func NovoGrupo(nome string) (*Grupo, error) {
	nome, err := textoObrigatorio(nome, masculino("nome do grupo"), tamanhoMaximoNomeGrupo)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &Grupo{
		ID:           uuid.New(),
		Nome:         nome,
		NomeBusca:    NomeDeBusca(nome),
		Ativo:        true,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

func (g *Grupo) Renomear(nome string, agora time.Time) error {
	nome, err := textoObrigatorio(nome, masculino("nome do grupo"), tamanhoMaximoNomeGrupo)
	if err != nil {
		return err
	}

	g.Nome = nome
	g.NomeBusca = NomeDeBusca(nome)
	g.AtualizadoEm = agora

	return nil
}

func (g *Grupo) Ativar(agora time.Time) {
	g.Ativo = true
	g.AtualizadoEm = agora
}

func (g *Grupo) Desativar(agora time.Time) {
	g.Ativo = false
	g.AtualizadoEm = agora
}
