package orders

import (
	"fmt"

	"github.com/yzimhao/trading_engine/cmd/haobase/assets"
	"github.com/yzimhao/trading_engine/cmd/haobase/base/varieties"
	"github.com/yzimhao/trading_engine/cmd/haomatch/matching"
	"github.com/yzimhao/trading_engine/trading_core"
	"github.com/yzimhao/trading_engine/utils"
	"github.com/yzimhao/trading_engine/utils/app"
)

func NewLimitOrder(user_id string, symbol string, side trading_core.OrderSide, price, qty string) (order *Order, err error) {
	return limit_order(user_id, symbol, side, price, qty)
}

func limit_order(user_id string, symbol string, side trading_core.OrderSide, price, qty string) (order *Order, err error) {
	varieties := varieties.NewTradingVarieties(symbol)
	if varieties.Id == 0 {
		return nil, fmt.Errorf("symbol %s invaild", symbol)
	}
	neworder := Order{
		OrderId:        generate_order_id_by_side(side),
		Symbol:         symbol,
		OrderSide:      side,
		OrderType:      trading_core.OrderTypeLimit,
		UserId:         user_id,
		Price:          price,
		AvgPrice:       "0",
		Quantity:       qty,
		Amount:         "0",
		FinishedQty:    "0",
		FeeRate:        string(varieties.FeeRate),
		FreezeAmount:   "0",
		FreezeQty:      "0",
		Fee:            "0",
		FinishedAmount: "0",
		Status:         OrderStatusNew,
	}
	unfinished := UnfinishedOrder{}

	db := app.Database().NewSession()
	defer db.Close()

	// 事务开启前创建可能需要的表
	if err := auto_create_table(symbol, varieties.Target.Symbol, varieties.Base.Symbol); err != nil {
		return nil, err
	}

	if _, err := order_pre_inspection(varieties, &neworder); err != nil {
		return nil, err
	}

	err = db.Begin()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			db.Rollback()
		} else {
			db.Commit()
		}
	}()

	//冻结相应资产
	if neworder.OrderSide == trading_core.OrderSideSell {
		//卖单部分fee在订单成交后结算的部分收取
		_, err = assets.FreezeAssets(db, user_id, varieties.Target.Symbol, neworder.Quantity, neworder.OrderId, assets.Behavior_Trade, neworder.Symbol)
		if err != nil {
			return nil, err
		}
		neworder.FreezeQty = neworder.Quantity
	} else if neworder.OrderSide == trading_core.OrderSideBuy {
		//买单的冻结金额加上手续费，这里预估全部成交所需要的手续费，
		amount := utils.D(neworder.Price).Mul(utils.D(neworder.Quantity))
		fee := amount.Mul(utils.D(neworder.FeeRate))
		freeze_amount := amount.Add(fee).String()

		//fee、tradeamount字段在结算程序中修改
		neworder.FreezeAmount = freeze_amount
		_, err = assets.FreezeAssets(db, user_id, varieties.Base.Symbol, freeze_amount, neworder.OrderId, assets.Behavior_Trade, neworder.Symbol)
		if err != nil {
			return nil, err
		}
	}

	unfinished.Order = neworder
	if err = neworder.Save(db); err != nil {
		return nil, err
	}

	unfinished.Order.CreateTime = neworder.CreateTime
	err = unfinished.Save(db)
	if err != nil {
		return nil, err
	}

	push_new_order_to_redis(neworder.Symbol, func() []byte {
		data := matching.Order{
			OrderId:   neworder.OrderId,
			OrderType: neworder.OrderType,
			Side:      neworder.OrderSide,
			Price:     neworder.Price,
			Qty:       neworder.Quantity,
			MaxQty:    "0",
			Amount:    "0",
			MaxAmount: "0",
			At:        neworder.CreateTime,
		}
		return data.Json()
	}())
	return &neworder, nil
}
