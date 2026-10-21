package divida

import (
	"fmt"
	"sort"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Renegociacao struct {
	Substituta        *Divida
	Parcelas          []*Parcela
	Contas            []ContaCancelada
	SaldoEmAberto     int64
	DetalheOriginal   string
	DetalheSubstituta string
}

func (d *Divida) Renegociar(motivo string, dados DadosDivida, usuarioID uuid.UUID, agora time.Time) (*Renegociacao, error) {
	if err := d.exigirAtiva(); err != nil {
		return nil, err
	}
	texto, err := textoObrigatorio(motivo, masculino("motivo da renegociação"), tamanhoMaximoMotivo)
	if err != nil {
		return nil, err
	}

	substituta, err := NovaDivida(d.EmpresaID, usuarioID, dados, agora)
	if err != nil {
		return nil, err
	}
	if substituta.DataSaldo.Before(d.DataSaldo) {
		return nil, domain.ErroValidacao("a data do saldo da renegociação não pode ser anterior à da dívida original, " +
			formatarData(d.DataSaldo))
	}
	origemID := d.ID
	substituta.DividaOrigemID = &origemID

	saldo := d.SaldoDevedor()
	parcelas, contas := d.encerrarParcelas(agora)
	d.Status = StatusRenegociada
	d.Motivo = &texto
	d.AtualizadoEm = agora

	return &Renegociacao{
		Substituta:    substituta,
		Parcelas:      parcelas,
		Contas:        contas,
		SaldoEmAberto: saldo,
		DetalheOriginal: fmt.Sprintf("renegociada: saldo em aberto %s, novo saldo %s; %s",
			formatarReais(saldo), formatarReais(substituta.SaldoInicialCentavos), texto),
		DetalheSubstituta: fmt.Sprintf("renegociação de \"%s\" (saldo em aberto %s): saldo devedor %s em %s, %d parcelas",
			d.Descricao, formatarReais(saldo), formatarReais(substituta.SaldoInicialCentavos),
			formatarData(substituta.DataSaldo), len(substituta.Parcelas)),
	}, nil
}

func (d *Divida) ReabrirParcela(parcelaID uuid.UUID, agora time.Time) (*Parcela, *Parcela, error) {
	if err := d.exigirAtiva(); err != nil {
		return nil, nil, err
	}
	antiga, err := d.Parcela(parcelaID)
	if err != nil {
		return nil, nil, err
	}
	if antiga.Status != StatusParcelaGerada {
		return nil, nil, domain.ErroConflito("só a parcela com contas a pagar geradas pode ser reaberta")
	}
	encerradas := 0
	for _, conta := range antiga.Contas() {
		if conta.aberta() {
			return nil, nil, domain.ErroConflito("cancele ou rejeite antes as contas a pagar da parcela no contas a pagar")
		}
		if conta.encerrada() {
			encerradas++
		}
	}
	if encerradas == 0 {
		return nil, nil, domain.ErroConflito("a parcela não tem conta a pagar cancelada ou rejeitada para reabrir")
	}

	nova := &Parcela{
		ID:                  uuid.Must(uuid.NewV7()),
		DividaID:            d.ID,
		Numero:              antiga.Numero,
		Vencimento:          antiga.Vencimento,
		AmortizacaoCentavos: antiga.AmortizacaoEmAberto(),
		JurosCentavos:       antiga.JurosEmAberto(),
		Status:              StatusParcelaPrevista,
		CriadoEm:            agora,
		AtualizadoEm:        agora,
	}
	antiga.cancelar(agora)
	antiga.SubstitutaID = &nova.ID
	d.Parcelas = append(d.Parcelas, nova)
	sort.SliceStable(d.Parcelas, func(i, j int) bool { return d.Parcelas[i].Numero < d.Parcelas[j].Numero })
	d.AtualizadoEm = agora

	return antiga, nova, nil
}

func (d *Divida) DetalheDaReabertura(antiga, nova *Parcela) string {
	return fmt.Sprintf("parcela %d reaberta: amortização %s e juros %s voltam a gerar conta a pagar",
		antiga.Numero, formatarReais(nova.AmortizacaoCentavos), formatarReais(nova.JurosCentavos))
}
