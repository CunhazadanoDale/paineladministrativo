package lead_test

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var (
	_ portsout.LeadRepository  = (*repositorioLeads)(nil)
	_ portsout.EtapaRepository = (*repositorioEtapas)(nil)
)

type movimentacao struct {
	leadID   uuid.UUID
	anterior uuid.UUID
	atual    uuid.UUID
}

type repositorioLeads struct {
	itens         map[uuid.UUID]*domainlead.Lead
	movimentacoes []movimentacao
}

func novoRepositorioLeads() *repositorioLeads {
	return &repositorioLeads{itens: map[uuid.UUID]*domainlead.Lead{}}
}

func (r *repositorioLeads) Criar(_ context.Context, item *domainlead.Lead) (uuid.UUID, error) {
	copia := *item
	r.itens[item.ID] = &copia

	return item.ID, nil
}

func (r *repositorioLeads) Atualizar(_ context.Context, item *domainlead.Lead) error {
	atual, ok := r.itens[item.ID]
	if !ok {
		return nil
	}

	copia := *item
	copia.EtapaID = atual.EtapaID
	r.itens[item.ID] = &copia

	return nil
}

func (r *repositorioLeads) Obter(_ context.Context, id uuid.UUID) (*domainlead.Lead, error) {
	item, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *item

	return &copia, nil
}

func (r *repositorioLeads) ListarPorFunil(_ context.Context, _ uuid.UUID) ([]*domainlead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) ListarPorEtapa(_ context.Context, _ uuid.UUID) ([]*domainlead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) ListarAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainlead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) Buscar(_ context.Context, _ string, _ domain.PaginacaoFiltro) ([]*domainlead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) Remover(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

func (r *repositorioLeads) ContarPorEtapa(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *repositorioLeads) ContarPorFunil(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *repositorioLeads) MoverParaEtapa(_ context.Context, leadID, anterior, atual uuid.UUID) error {
	item, ok := r.itens[leadID]
	if !ok || item.EtapaID != anterior {
		return domain.ErroConflito("o lead mudou de etapa durante a operação")
	}

	item.EtapaID = atual
	r.movimentacoes = append(r.movimentacoes, movimentacao{leadID: leadID, anterior: anterior, atual: atual})

	return nil
}

type repositorioEtapas struct {
	itens map[uuid.UUID]*domainlead.Etapa
}

func novoRepositorioEtapas() *repositorioEtapas {
	return &repositorioEtapas{itens: map[uuid.UUID]*domainlead.Etapa{}}
}

func (r *repositorioEtapas) Criar(_ context.Context, etapa *domainlead.Etapa) (uuid.UUID, error) {
	copia := *etapa
	r.itens[etapa.EtapaID] = &copia

	return etapa.EtapaID, nil
}

func (r *repositorioEtapas) Atualizar(_ context.Context, _ *domainlead.Etapa) error {
	return nil
}

func (r *repositorioEtapas) Obter(_ context.Context, id uuid.UUID) (*domainlead.Etapa, error) {
	etapa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *etapa

	return &copia, nil
}

func (r *repositorioEtapas) ListarPorFunil(_ context.Context, funilID uuid.UUID) ([]*domainlead.Etapa, error) {
	var etapas []*domainlead.Etapa
	for _, etapa := range r.itens {
		if etapa.FunilID == funilID {
			copia := *etapa
			etapas = append(etapas, &copia)
		}
	}

	return etapas, nil
}

func (r *repositorioEtapas) ListarPorFunilOrdenado(_ context.Context, _ uuid.UUID) ([]*domainlead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) Remover(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *repositorioEtapas) Reordenar(_ context.Context, _ uuid.UUID, etapas []*domainlead.Etapa) error {
	for i, etapa := range etapas {
		r.itens[etapa.EtapaID].Ordem = i + 1
	}

	return nil
}

func (r *repositorioEtapas) ObterProxima(_ context.Context, _ uuid.UUID) (*domainlead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) ObterAnterior(_ context.Context, _ uuid.UUID) (*domainlead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) ExistePorFunil(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
