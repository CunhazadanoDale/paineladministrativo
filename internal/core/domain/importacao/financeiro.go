package importacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

func ErroContasAPagarAbertas() error {
	return domain.ErroConflito("cancele ou rejeite antes as contas a pagar geradas pela importação")
}

func (i *Importacao) exigirLancamentoFinanceiro() error {
	if i.Status == StatusCancelada {
		return domain.ErroConflito("importação cancelada não recebe parcela nem custo")
	}

	return nil
}

func (i *Importacao) Parcela(id uuid.UUID) *Parcela {
	for _, parcela := range i.Parcelas {
		if parcela.ID == id {
			return parcela
		}
	}

	return nil
}

func (i *Importacao) Custo(id uuid.UUID) *Custo {
	for _, custo := range i.Custos {
		if custo.ID == id {
			return custo
		}
	}

	return nil
}

func (i *Importacao) IncluirParcela(usuarioID uuid.UUID, dados DadosParcela, agora time.Time) (*Parcela, error) {
	if err := i.exigirLancamentoFinanceiro(); err != nil {
		return nil, err
	}

	parcela := &Parcela{
		ID:           uuid.Must(uuid.NewV7()),
		ImportacaoID: i.ID,
		Status:       StatusParcelaAberta,
		UsuarioID:    usuarioID,
		CriadoEm:     agora,
	}
	if err := parcela.aplicarDados(dados, agora); err != nil {
		return nil, err
	}
	if err := i.conferirParcelasNoTotal(i.ValorTotal, parcela.Valor, uuid.Nil); err != nil {
		return nil, err
	}

	i.Parcelas = append(i.Parcelas, parcela)
	i.AtualizadoEm = agora

	return parcela, nil
}

func (i *Importacao) AlterarParcela(id uuid.UUID, dados DadosParcela, agora time.Time) (*Parcela, error) {
	parcela, err := i.parcelaAberta(id)
	if err != nil {
		return nil, err
	}

	alterada := *parcela
	if err := alterada.aplicarDados(dados, agora); err != nil {
		return nil, err
	}
	if err := i.conferirParcelasNoTotal(i.ValorTotal, alterada.Valor, id); err != nil {
		return nil, err
	}

	*parcela = alterada
	i.AtualizadoEm = agora

	return parcela, nil
}

func (i *Importacao) RemoverParcela(id uuid.UUID, agora time.Time) (*Parcela, error) {
	parcela, err := i.parcelaAberta(id)
	if err != nil {
		return nil, err
	}

	restantes := make([]*Parcela, 0, len(i.Parcelas)-1)
	for _, outra := range i.Parcelas {
		if outra.ID != id {
			restantes = append(restantes, outra)
		}
	}
	i.Parcelas = restantes
	i.AtualizadoEm = agora

	return parcela, nil
}

func (i *Importacao) FecharCambio(id uuid.UUID, input FechamentoCambioInput, agora time.Time) (*Parcela, error) {
	parcela := i.Parcela(id)
	if parcela == nil {
		return nil, domain.ErroNaoEncontrado("parcela não encontrada")
	}
	if err := i.exigirLancamentoFinanceiro(); err != nil {
		return nil, err
	}

	cambio, err := i.cambioDaParcela(input.CambioMicro)
	if err != nil {
		return nil, err
	}
	if err := parcela.fecharCambio(cambio, input.Vencimento, agora); err != nil {
		return nil, err
	}
	i.AtualizadoEm = agora

	return parcela, nil
}

func (i *Importacao) ReabrirParcela(id, usuarioID uuid.UUID, agora time.Time) (*Parcela, *Parcela, error) {
	parcela := i.Parcela(id)
	if parcela == nil {
		return nil, nil, domain.ErroNaoEncontrado("parcela não encontrada")
	}
	if err := i.exigirLancamentoFinanceiro(); err != nil {
		return nil, nil, err
	}
	if parcela.Status != StatusParcelaCambioFechado {
		return nil, nil, domain.ErroConflito("só a parcela com câmbio fechado pode ser reaberta")
	}
	if !parcela.contaEncerrada() {
		return nil, nil, domain.ErroConflito("cancele ou rejeite antes a conta a pagar da parcela")
	}

	parcela.Status = StatusParcelaCancelada
	parcela.AtualizadoEm = agora
	nova := &Parcela{
		ID:                 uuid.Must(uuid.NewV7()),
		ImportacaoID:       i.ID,
		Descricao:          parcela.Descricao,
		Valor:              parcela.Valor,
		VencimentoPrevisto: parcela.VencimentoPrevisto,
		Status:             StatusParcelaAberta,
		UsuarioID:          usuarioID,
		CriadoEm:           agora,
		AtualizadoEm:       agora,
	}
	i.Parcelas = append(i.Parcelas, nova)
	i.AtualizadoEm = agora

	return parcela, nova, nil
}

func (i *Importacao) LancarCusto(usuarioID uuid.UUID, dados DadosCusto, agora time.Time) (*Custo, error) {
	if err := i.exigirLancamentoFinanceiro(); err != nil {
		return nil, err
	}

	custo, err := novoCusto(i, usuarioID, dados, agora)
	if err != nil {
		return nil, err
	}
	i.Custos = append(i.Custos, custo)
	i.AtualizadoEm = agora

	return custo, nil
}

func (i *Importacao) CancelarCusto(id, usuarioID uuid.UUID, agora time.Time) (*Custo, error) {
	custo := i.Custo(id)
	if custo == nil {
		return nil, domain.ErroNaoEncontrado("custo não encontrado")
	}
	switch {
	case !custo.Ativo():
		return nil, domain.ErroConflito("o custo já foi cancelado")
	case i.Status == StatusRecebida && custo.CompoeCusto:
		return nil, domain.ErroConflito("o custo já entrou no custo dos lotes recebidos e não pode ser cancelado")
	case !custo.contaEncerrada():
		return nil, domain.ErroConflito("cancele ou rejeite antes a conta a pagar do custo")
	}

	momento := agora
	usuario := usuarioID
	custo.CanceladoEm = &momento
	custo.CanceladoPor = &usuario
	i.AtualizadoEm = agora

	return custo, nil
}

func (i *Importacao) ContasAPagarAbertas() []uuid.UUID {
	abertas := make([]uuid.UUID, 0)
	for _, parcela := range i.Parcelas {
		if parcela.Status == StatusParcelaCambioFechado && !parcela.contaEncerrada() {
			abertas = append(abertas, *parcela.SolicitacaoID)
		}
	}
	for _, custo := range i.Custos {
		if custo.Ativo() && !custo.contaEncerrada() {
			abertas = append(abertas, custo.SolicitacaoID)
		}
	}

	return abertas
}

func (i *Importacao) ContasAPagarVinculadas() []uuid.UUID {
	vinculadas := make([]uuid.UUID, 0)
	for _, parcela := range i.Parcelas {
		if parcela.Status == StatusParcelaCambioFechado {
			vinculadas = append(vinculadas, *parcela.SolicitacaoID)
		}
	}
	for _, custo := range i.Custos {
		if custo.Ativo() {
			vinculadas = append(vinculadas, custo.SolicitacaoID)
		}
	}

	return vinculadas
}

func (i *Importacao) SomaParcelasAtivas() ValorMoeda {
	var soma ValorMoeda
	for _, parcela := range i.Parcelas {
		if parcela.Ativa() {
			soma = ValorMoedaDe(soma.Centesimos() + parcela.Valor.Centesimos())
		}
	}

	return soma
}

func (i *Importacao) parcelaAberta(id uuid.UUID) (*Parcela, error) {
	parcela := i.Parcela(id)
	if parcela == nil {
		return nil, domain.ErroNaoEncontrado("parcela não encontrada")
	}
	if err := i.exigirLancamentoFinanceiro(); err != nil {
		return nil, err
	}
	if parcela.Status != StatusParcelaAberta {
		return nil, domain.ErroConflito("só a parcela sem câmbio fechado pode ser alterada ou removida")
	}

	return parcela, nil
}

func (i *Importacao) conferirParcelasNoTotal(total, nova ValorMoeda, ignorar uuid.UUID) error {
	soma := nova.Centesimos()
	for _, parcela := range i.Parcelas {
		if parcela.Ativa() && parcela.ID != ignorar {
			soma += parcela.Valor.Centesimos()
		}
	}
	if soma > total.Centesimos() {
		return domain.ErroValidacao("a soma das parcelas não pode passar do valor dos itens da importação")
	}

	return nil
}

func (i *Importacao) cambioDaParcela(micro *int64) (Cambio, error) {
	if i.Moeda == domainfornecedor.MoedaReal {
		if micro != nil && *micro != cambioMicroDoReal {
			return Cambio{}, domain.ErroValidacao("importação em reais usa câmbio 1,000000")
		}

		return CambioDoReal(), nil
	}
	if micro == nil {
		return Cambio{}, domain.ErroValidacao(masculino("câmbio").obrigatorio())
	}

	return NovoCambio(*micro)
}
