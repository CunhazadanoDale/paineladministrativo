package empresa_test

import (
	"testing"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

func TestAcessoTotalPermiteQualquerEmpresa(t *testing.T) {
	acesso := domainempresa.AcessoTotal()

	if !acesso.Todas() || !acesso.Permite(uuid.New()) {
		t.Error("acesso total deveria permitir qualquer empresa")
	}
	if acesso.EmpresaIDs() != nil {
		t.Errorf("acesso total devolveu lista %v, esperado nil", acesso.EmpresaIDs())
	}
}

func TestAcessoRestritoPermiteSoAsVinculadas(t *testing.T) {
	primeira, segunda, fora := uuid.New(), uuid.New(), uuid.New()
	acesso := domainempresa.AcessoRestrito([]uuid.UUID{primeira, segunda, primeira})

	if acesso.Todas() {
		t.Fatal("acesso restrito marcado como total")
	}
	if !acesso.Permite(primeira) || !acesso.Permite(segunda) || acesso.Permite(fora) {
		t.Error("acesso restrito permitiu ou negou empresa errada")
	}
	if ids := acesso.EmpresaIDs(); len(ids) != 2 {
		t.Errorf("EmpresaIDs = %v, esperado as duas empresas sem repetição", ids)
	}
}

func TestAcessoSemEmpresasNaoPermiteNadaENaoViraTotal(t *testing.T) {
	acesso := domainempresa.AcessoRestrito(nil)

	if acesso.Todas() || acesso.Permite(uuid.New()) {
		t.Error("acesso vazio deveria negar tudo")
	}
	if ids := acesso.EmpresaIDs(); ids == nil || len(ids) != 0 {
		t.Errorf("EmpresaIDs = %#v, esperado lista vazia não nula para não virar 'sem filtro'", ids)
	}
}
