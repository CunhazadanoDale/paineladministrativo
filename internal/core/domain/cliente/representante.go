package cliente

import (
	"time"

	"github.com/google/uuid"
)

const tamanhoMaximoNomeRepresentante = 160

type Representante struct {
	ID                    uuid.UUID
	Nome                  string
	NomeBusca             string
	Documento             *Documento
	UsuarioID             *uuid.UUID
	Email                 *string
	Telefone              *string
	ComissaoProntaEntrega Percentual
	ComissaoImportacao    Percentual
	Ativo                 bool
	CriadoEm              time.Time
	AtualizadoEm          time.Time
}

type DadosRepresentante struct {
	Nome                  string
	Documento             *Documento
	UsuarioID             *uuid.UUID
	Email                 *string
	Telefone              *string
	ComissaoProntaEntrega Percentual
	ComissaoImportacao    Percentual
}

func NovoRepresentante(dados DadosRepresentante) (*Representante, error) {
	representante := &Representante{ID: uuid.New(), Ativo: true}
	agora := time.Now().UTC()
	representante.CriadoEm = agora

	if err := representante.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return representante, nil
}

func (r *Representante) Alterar(dados DadosRepresentante, agora time.Time) error {
	nome, err := textoObrigatorio(dados.Nome, masculino("nome do representante"), tamanhoMaximoNomeRepresentante)
	if err != nil {
		return err
	}
	email, err := emailOpcional(dados.Email, "e-mail do representante")
	if err != nil {
		return err
	}
	telefone, err := textoOpcional(dados.Telefone, "telefone do representante", tamanhoMaximoTelefone)
	if err != nil {
		return err
	}
	if dados.Documento != nil && dados.Documento.Vazio() {
		dados.Documento = nil
	}
	if dados.UsuarioID != nil && *dados.UsuarioID == uuid.Nil {
		dados.UsuarioID = nil
	}

	r.Nome = nome
	r.NomeBusca = NomeDeBusca(nome)
	r.Documento = dados.Documento
	r.UsuarioID = dados.UsuarioID
	r.Email = email
	r.Telefone = telefone
	r.ComissaoProntaEntrega = dados.ComissaoProntaEntrega
	r.ComissaoImportacao = dados.ComissaoImportacao
	r.AtualizadoEm = agora

	return nil
}

func (r *Representante) Ativar(agora time.Time) {
	r.Ativo = true
	r.AtualizadoEm = agora
}

func (r *Representante) Desativar(agora time.Time) {
	r.Ativo = false
	r.AtualizadoEm = agora
}
