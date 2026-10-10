package leadpoint

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoNomeLead     = 200
	tamanhoMaximoEmailLead    = 255
	tamanhoMaximoTelefoneLead = 40
	tamanhoMaximoOrigemLead   = 80
)

var _ portsin.LeadUseCase = (*LeadUsecaseImpl)(nil)

type LeadUsecaseImpl struct {
	repo   portsout.LeadRepository
	etapas portsout.EtapaRepository
}

func NewLeadUsecase(repo portsout.LeadRepository, etapas portsout.EtapaRepository) *LeadUsecaseImpl {
	return &LeadUsecaseImpl{repo: repo, etapas: etapas}
}

func (l *LeadUsecaseImpl) Create(ctx context.Context, lead *lead.Lead) (uuid.UUID, error) {
	if lead == nil {
		return uuid.Nil, domain.ErroValidacao("lead não informado")
	}

	normalizarLead(lead)

	if err := validarCadastro(lead); err != nil {
		return uuid.Nil, err
	}
	if lead.EtapaID == uuid.Nil {
		return uuid.Nil, domain.ErroValidacao("etapa do lead é obrigatória")
	}
	if _, err := l.etapaAtiva(ctx, lead.EtapaID, "etapa do lead"); err != nil {
		return uuid.Nil, err
	}
	if lead.ID == uuid.Nil {
		lead.ID = uuid.New()
	}

	agora := time.Now().UTC()
	if lead.CriadoEm.IsZero() {
		lead.CriadoEm = agora
	}
	lead.AtualizadoEm = agora
	lead.Ativo = true

	return l.repo.Create(ctx, lead)
}

func (l *LeadUsecaseImpl) Update(ctx context.Context, lead *lead.Lead) error {
	if lead == nil || lead.ID == uuid.Nil {
		return domain.ErroValidacao("lead inválido")
	}

	normalizarLead(lead)

	if err := validarCadastro(lead); err != nil {
		return err
	}

	atual, err := l.buscar(ctx, lead.ID)
	if err != nil {
		return err
	}
	if lead.EtapaID != uuid.Nil && lead.EtapaID != atual.EtapaID {
		return domain.ErroValidacao("a etapa do lead só muda pela movimentação de etapa")
	}

	lead.EtapaID = atual.EtapaID
	lead.CriadoEm = atual.CriadoEm
	lead.AtualizadoEm = time.Now().UTC()

	return l.repo.Update(ctx, lead)
}

func (l *LeadUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*lead.Lead, error) {
	return l.buscar(ctx, id)
}

func (l *LeadUsecaseImpl) ListByFunil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error) {
	if funilID == uuid.Nil {
		return nil, domain.ErroValidacao("funil não informado")
	}

	return l.repo.ListByFunil(ctx, funilID)
}

func (l *LeadUsecaseImpl) ListByEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error) {
	if etapaID == uuid.Nil {
		return nil, domain.ErroValidacao("etapa não informada")
	}

	return l.repo.ListByEtapa(ctx, etapaID)
}

func (l *LeadUsecaseImpl) ListAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return l.repo.ListAtivos(ctx, paginacao.Normalizada())
}

func (l *LeadUsecaseImpl) Search(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return l.repo.Search(ctx, strings.TrimSpace(query), paginacao.Normalizada())
}

func (l *LeadUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := l.buscar(ctx, id); err != nil {
		return err
	}

	return l.repo.Delete(ctx, id)
}

func (l *LeadUsecaseImpl) CountByEtapa(ctx context.Context, etapaID uuid.UUID) (int, error) {
	if etapaID == uuid.Nil {
		return 0, domain.ErroValidacao("etapa não informada")
	}

	return l.repo.CountByEtapa(ctx, etapaID)
}

func (l *LeadUsecaseImpl) CountByFunil(ctx context.Context, funilID uuid.UUID) (int, error) {
	if funilID == uuid.Nil {
		return 0, domain.ErroValidacao("funil não informado")
	}

	return l.repo.CountByFunil(ctx, funilID)
}

func (l *LeadUsecaseImpl) UpdateEtapa(ctx context.Context, leadID uuid.UUID, newEtapaID uuid.UUID) error {
	if leadID == uuid.Nil {
		return domain.ErroValidacao("lead não informado")
	}
	if newEtapaID == uuid.Nil {
		return domain.ErroValidacao("etapa de destino não informada")
	}

	atual, err := l.buscar(ctx, leadID)
	if err != nil {
		return err
	}
	if atual.EtapaID == newEtapaID {
		return nil
	}

	destino, err := l.etapaAtiva(ctx, newEtapaID, "etapa de destino")
	if err != nil {
		return err
	}

	origem, err := l.etapas.GetByID(ctx, atual.EtapaID)
	if err != nil {
		return err
	}
	if origem != nil && origem.FunilID != destino.FunilID {
		return domain.ErroValidacao("a etapa de destino pertence a outro funil")
	}

	return l.repo.MoverParaEtapa(ctx, leadID, atual.EtapaID, newEtapaID)
}

func (l *LeadUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*lead.Lead, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do lead não informado")
	}

	item, err := l.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (l *LeadUsecaseImpl) etapaAtiva(ctx context.Context, etapaID uuid.UUID, descricao string) (*lead.Etapa, error) {
	etapa, err := l.etapas.GetByID(ctx, etapaID)
	if err != nil {
		return nil, err
	}
	if etapa == nil {
		return nil, domain.ErroValidacao(descricao + " não encontrada")
	}
	if !etapa.Ativo {
		return nil, domain.ErroValidacao(descricao + " está inativa")
	}

	return etapa, nil
}

func normalizarLead(lead *lead.Lead) {
	lead.Nome = strings.TrimSpace(lead.Nome)
	lead.Email = strings.ToLower(strings.TrimSpace(lead.Email))
	lead.Telefone = strings.TrimSpace(lead.Telefone)
	lead.Origem = strings.TrimSpace(lead.Origem)
}

func validarCadastro(lead *lead.Lead) error {
	if lead.Nome == "" {
		return domain.ErroValidacao("nome do lead é obrigatório")
	}
	if err := validarTamanho("nome do lead", lead.Nome, tamanhoMaximoNomeLead); err != nil {
		return err
	}
	if err := validarTamanho("email do lead", lead.Email, tamanhoMaximoEmailLead); err != nil {
		return err
	}
	if lead.Email != "" && !emailValido(lead.Email) {
		return domain.ErroValidacao("email do lead é inválido")
	}
	if err := validarTamanho("telefone do lead", lead.Telefone, tamanhoMaximoTelefoneLead); err != nil {
		return err
	}

	return validarTamanho("origem do lead", lead.Origem, tamanhoMaximoOrigemLead)
}

func validarTamanho(campo, valor string, maximo int) error {
	if utf8.RuneCountInString(valor) > maximo {
		return domain.ErroValidacao(fmt.Sprintf("%s deve ter no máximo %d caracteres", campo, maximo))
	}

	return nil
}

func emailValido(email string) bool {
	local, dominio, achou := strings.Cut(email, "@")

	return achou && local != "" && dominio != "" && !strings.Contains(dominio, "@") && !strings.ContainsAny(email, " \t")
}
