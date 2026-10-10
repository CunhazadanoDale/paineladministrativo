package lead

import (
	"context"
	"fmt"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.EtapaUseCase = (*EtapaUsecaseImpl)(nil)

type EtapaUsecaseImpl struct {
	repo portsout.EtapaRepository
}

func NewEtapaUsecase(repo portsout.EtapaRepository) *EtapaUsecaseImpl {
	return &EtapaUsecaseImpl{repo: repo}
}

func (e *EtapaUsecaseImpl) Criar(ctx context.Context, etapa *domainlead.Etapa) (uuid.UUID, error) {
	if etapa == nil {
		return uuid.Nil, domain.ErroValidacao("etapa não informada")
	}

	etapa.Nome = strings.TrimSpace(etapa.Nome)

	if etapa.Nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome da etapa é obrigatório")
	}
	if etapa.FunilID == uuid.Nil {
		return uuid.Nil, domain.ErroValidacao("funil da etapa é obrigatório")
	}
	if etapa.EtapaID == uuid.Nil {
		etapa.EtapaID = uuid.New()
	}

	if etapa.Ordem <= 0 {
		etapas, err := e.repo.ListarPorFunilOrdenado(ctx, etapa.FunilID)
		if err != nil {
			return uuid.Nil, err
		}
		etapa.Ordem = len(etapas) + 1
	}

	etapa.Ativo = true

	return e.repo.Criar(ctx, etapa)
}

func (e *EtapaUsecaseImpl) Atualizar(ctx context.Context, etapa *domainlead.Etapa) error {
	if etapa == nil || etapa.EtapaID == uuid.Nil {
		return domain.ErroValidacao("etapa inválida")
	}

	etapa.Nome = strings.TrimSpace(etapa.Nome)

	if etapa.Nome == "" {
		return domain.ErroValidacao("nome da etapa é obrigatório")
	}
	if etapa.FunilID == uuid.Nil {
		return domain.ErroValidacao("funil da etapa é obrigatório")
	}
	if etapa.Ordem <= 0 {
		return domain.ErroValidacao("ordem da etapa é obrigatória")
	}

	atual, err := e.repo.Obter(ctx, etapa.EtapaID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return e.repo.Atualizar(ctx, etapa)
}

func (e *EtapaUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainlead.Etapa, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id da etapa não informado")
	}

	item, err := e.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (e *EtapaUsecaseImpl) ListarPorFunil(ctx context.Context, funilID uuid.UUID) ([]*domainlead.Etapa, error) {
	if funilID == uuid.Nil {
		return nil, domain.ErroValidacao("funil não informado")
	}

	return e.repo.ListarPorFunil(ctx, funilID)
}

func (e *EtapaUsecaseImpl) ListarPorFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*domainlead.Etapa, error) {
	if funilID == uuid.Nil {
		return nil, domain.ErroValidacao("funil não informado")
	}

	return e.repo.ListarPorFunilOrdenado(ctx, funilID)
}

func (e *EtapaUsecaseImpl) Remover(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErroValidacao("id da etapa não informado")
	}

	atual, err := e.repo.Obter(ctx, id)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return e.repo.Remover(ctx, id)
}

func (e *EtapaUsecaseImpl) Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*domainlead.Etapa) error {
	if funilID == uuid.Nil {
		return domain.ErroValidacao("funil não informado")
	}
	if len(etapas) == 0 {
		return domain.ErroValidacao("nenhuma etapa informada para reordenar")
	}
	for i, etapa := range etapas {
		if etapa == nil {
			return fmt.Errorf("%w: etapa inválida na posição %d", domain.ErrValidacao, i)
		}
		if etapa.FunilID != funilID {
			return domain.ErroValidacao("etapa " + etapa.Nome + " não pertence ao funil informado")
		}
	}

	atuais, err := e.repo.ListarPorFunil(ctx, funilID)
	if err != nil {
		return err
	}
	if !mesmasEtapas(atuais, etapas) {
		return domain.ErroValidacao("a reordenação precisa trazer todas as etapas do funil, sem repetição")
	}

	return e.repo.Reordenar(ctx, funilID, etapas)
}

func (e *EtapaUsecaseImpl) ObterProxima(ctx context.Context, currentEtapaID uuid.UUID) (*domainlead.Etapa, error) {
	if currentEtapaID == uuid.Nil {
		return nil, domain.ErroValidacao("etapa atual não informada")
	}

	return e.repo.ObterProxima(ctx, currentEtapaID)
}

func (e *EtapaUsecaseImpl) ObterAnterior(ctx context.Context, currentEtapaID uuid.UUID) (*domainlead.Etapa, error) {
	if currentEtapaID == uuid.Nil {
		return nil, domain.ErroValidacao("etapa atual não informada")
	}

	return e.repo.ObterAnterior(ctx, currentEtapaID)
}

func (e *EtapaUsecaseImpl) ExistePorFunil(ctx context.Context, funilID uuid.UUID) (bool, error) {
	if funilID == uuid.Nil {
		return false, domain.ErroValidacao("funil não informado")
	}

	return e.repo.ExistePorFunil(ctx, funilID)
}

func mesmasEtapas(atuais, informadas []*domainlead.Etapa) bool {
	if len(atuais) != len(informadas) {
		return false
	}

	pendentes := make(map[uuid.UUID]struct{}, len(atuais))
	for _, etapa := range atuais {
		pendentes[etapa.EtapaID] = struct{}{}
	}

	for _, etapa := range informadas {
		if _, ok := pendentes[etapa.EtapaID]; !ok {
			return false
		}
		delete(pendentes, etapa.EtapaID)
	}

	return true
}
