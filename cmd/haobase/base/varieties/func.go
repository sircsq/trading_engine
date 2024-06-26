package varieties

import (
	"github.com/yzimhao/trading_engine/types"
	"github.com/yzimhao/trading_engine/utils/app"
)

func NewVarieties(symbol string) *Varieties {
	db := app.Database().NewSession()
	defer db.Close()

	var row Varieties
	db.Where("symbol=?", symbol).Get(&row)
	return &row
}

func NewTradingVarieties(symbol string) *TradingVarieties {
	db := app.Database().NewSession()
	defer db.Close()

	var row TradingVarieties

	db.Where("symbol=?", symbol).Get(&row)
	if row.Id > 0 {
		row.Target = *newVarietiesById(row.TargetSymbolId)
		row.Base = *newVarietiesById(row.BaseSymbolId)
	}
	return &row
}

func AllTradingVarieties() []TradingVarieties {
	db := app.Database().NewSession()
	defer db.Close()

	var rows []TradingVarieties

	db.Table(new(TradingVarieties)).Where("status=?", types.StatusEnabled).OrderBy("sort asc, id desc").Find(&rows)
	for i, item := range rows {
		rows[i].Target = *newVarietiesById(item.TargetSymbolId)
		rows[i].Base = *newVarietiesById(item.BaseSymbolId)
	}
	return rows
}

func newVarietiesById(id int) *Varieties {
	db := app.Database().NewSession()
	defer db.Close()

	var row Varieties
	db.Where("id=?", id).Get(&row)
	return &row
}

func AllVarieties() []Varieties {
	db := app.Database().NewSession()
	defer db.Close()

	var rows []Varieties

	db.Table(new(Varieties)).OrderBy("id asc").Find(&rows)
	return rows
}
