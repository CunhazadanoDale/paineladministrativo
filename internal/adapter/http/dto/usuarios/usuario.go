package usuarios

import (
	"time"

	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type CriarUsuarioRequest struct {
	Nome    string    `json:"nome"`
	Email   string    `json:"email"`
	Senha   string    `json:"senha"`
	CargoID uuid.UUID `json:"cargo_id"`
}

type AtualizarUsuarioRequest struct {
	Nome    string    `json:"nome"`
	Email   string    `json:"email"`
	CargoID uuid.UUID `json:"cargo_id"`
	Ativo   bool      `json:"ativo"`
}

type AutenticarUsuarioRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type TrocarSenhaRequest struct {
	NovaSenha string `json:"nova_senha"`
}

type TrocarSenhaPropriaRequest struct {
	SenhaAtual string `json:"senha_atual"`
	NovaSenha  string `json:"nova_senha"`
}

type UsuarioResponse struct {
	ID           uuid.UUID `json:"id"`
	Nome         string    `json:"nome"`
	Email        string    `json:"email"`
	Ativo        bool      `json:"ativo"`
	CargoID      uuid.UUID `json:"cargo_id"`
	UltimoLogin  time.Time `json:"ultimo_login"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

type SessaoResponse struct {
	Token         string          `json:"token"`
	ExpiraEm      time.Time       `json:"expira_em"`
	Administrador bool            `json:"administrador"`
	Usuario       UsuarioResponse `json:"usuario"`
}

func (r AtualizarUsuarioRequest) ParaUsuario(id uuid.UUID) *domainusuarios.Usuario {
	return &domainusuarios.Usuario{
		ID:      id,
		Nome:    r.Nome,
		Email:   r.Email,
		CargoID: r.CargoID,
		Ativo:   r.Ativo,
	}
}

func NovaUsuarioResponse(item *domainusuarios.Usuario) UsuarioResponse {
	return UsuarioResponse{
		ID:           item.ID,
		Nome:         item.Nome,
		Email:        item.Email,
		Ativo:        item.Ativo,
		CargoID:      item.CargoID,
		UltimoLogin:  item.UltimoLogin,
		CriadoEm:     item.CriadoEm,
		AtualizadoEm: item.AtualizadoEm,
	}
}

func NovaUsuarioResponses(itens []*domainusuarios.Usuario) []UsuarioResponse {
	respostas := make([]UsuarioResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaUsuarioResponse(item))
	}

	return respostas
}
