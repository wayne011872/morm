package main

import (
	"encoding/json"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ShapeType string

func (p ShapeType) String() string {
	return string(p)
}

type ProductType string

func (p ProductType) String() string {
	return string(p)
}

type StockProductState string

func (p StockProductState) String() string {
	return string(p)
}

const (
	ShapeType_Alone     = ShapeType("alone")     // 商品形狀：單獨包裝
	ShapeType_BUCKET    = ShapeType("bucket")    // 商品形狀：桶類
	ShapeType_IRREGULAR = ShapeType("irregular") // 商品形狀：不規則狀
	ShapeType_LONG      = ShapeType("long")      // 商品形狀：長條狀
	ShapeType_SOLID     = ShapeType("solid")     // 商品形狀：固定形狀
	ShapeType_WASH      = ShapeType("wash")      // 商品形狀：洗劑消毒類

	ProductType_Basic      = ProductType("basic")
	ProductType_Formula    = ProductType("formula")
	ProductType_Collection = ProductType("collection")

	StockProductState_Supply = StockProductState("supply") // 庫存狀態：正常供應
	StockProductState_Stop   = StockProductState("stop")   //庫存狀態：停止供應
)

type PolyStockProduct struct {
	StockProck
}

func (f *PolyStockProduct) ToStockProductMetrial() *StockProductMaterial {
	if f.StockProck == nil || f.GetType() != ProductType_Basic {
		return nil
	}
	return f.StockProck.(*StockProductMaterial)
}

func (f *PolyStockProduct) ToStockProductFormulas() *StockProductFormulas {
	if f.StockProck == nil || f.GetType() != ProductType_Formula {
		return nil
	}
	return f.StockProck.(*StockProductFormulas)
}

func (f *PolyStockProduct) unmarshal(b []byte, myfunc func(data []byte, v interface{}) error) error {
	var v CommonStockProduct
	err := myfunc(b, &v)
	if err != nil {
		return err
	}
	var i StockProck
	switch v.GetType() {
	case ProductType_Basic:
		i = &StockProductMaterial{
			CommonStockProduct: &v,
		}
	case ProductType_Formula:
		i = &StockProductFormulas{
			CommonStockProduct: &v,
		}
	default:
		return errors.New("unknown log type: " + v.GetType().String())
	}
	err = myfunc(b, i)
	if err != nil {
		return err
	}
	f.StockProck = i
	return nil
}
func (f *PolyStockProduct) UnmarshalBSON(b []byte) error {
	return f.unmarshal(b, bson.Unmarshal)
}

func (f *PolyStockProduct) UnmarshalJSON(b []byte) error {
	return f.unmarshal(b, json.Unmarshal)
}

func (f *PolyStockProduct) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.StockProck)
}

func (f *PolyStockProduct) MarshalBSON() ([]byte, error) {
	return bson.Marshal(f.StockProck)
}

type StockProck interface {
	GetType() ProductType
	GetShape() ShapeType
	GetName() string
	GetState() StockProductState
	Validate() error
	GetUpdateFields() bson.D
}

// 要叫StockProduct，不能叫StockMaterial，因為這裡面包含組合商品
type CommonStockProduct struct {
	Name         string            // 庫存商品名稱
	Typ          ProductType       // 商品種類：一般商品、組合商品
	Volume       *Volume           // 材積
	Shape        ShapeType         // 型狀
	BucketHeight float64           // 每增加一桶的高度
	Weight       float64           // 權重
	State        StockProductState // 狀態
}

func (s *CommonStockProduct) GetName() string {
	return s.Name
}

func (s *CommonStockProduct) GetType() ProductType {
	return s.Typ
}

func (s *CommonStockProduct) GetShape() ShapeType {
	return s.Shape
}

func (s *CommonStockProduct) GetState() StockProductState {
	return s.State
}

func (s *CommonStockProduct) GetUpdateFields() bson.D {
	return bson.D{
		{Key: "polystockproduct.name", Value: s.Name},
		{Key: "polystockproduct.typ", Value: s.Typ},
		{Key: "polystockproduct.volume", Value: s.Volume},
		{Key: "polystockproduct.shape", Value: s.Shape},
		{Key: "polystockproduct.bucketheight", Value: s.BucketHeight},
		{Key: "polystockproduct.weight", Value: s.Weight},
		{Key: "polystockproduct.state", Value: s.State},
	}
}

func (s *CommonStockProduct) Validate() error {
	if s.Name == "" {
		return errors.New("請輸入庫存商品名稱")
	}

	if s.Typ != "basic" && s.Typ != "formula" {
		return errors.New("錯誤商品種類")
	}

	if s.Volume == nil {
		return errors.New("請填入材積")
	}

	if s.Volume.Height < 0 {
		return errors.New("材積的高不得小於0")
	}

	if s.Volume.Length < 0 {
		return errors.New("材積的長不得小於0")
	}

	if s.Volume.Width < 0 {
		return errors.New("材積的寬不得小於0")
	}

	if s.Shape != "alone" && s.Shape != "bucket" && s.Shape != "irregular" &&
		s.Shape != "long" && s.Shape != "solid" && s.Shape != "wash" {
		return errors.New("錯誤形狀")
	}

	if s.Shape == "bucket" {
		if s.BucketHeight < 0 {
			return errors.New("每增加一桶的高度設定不得小於0")
		}
	}

	if s.Shape != "bucket" {
		if s.BucketHeight > 0 {
			return errors.New("不得填寫每增加一桶的高度設定")
		}
	}

	if s.BucketHeight < 0 {
		return errors.New("每增加一桶的高度設定不得下小於0")
	}

	if s.Weight < 0 {
		return errors.New("權重設定不得小於0")
	}

	if s.State != "supply" && s.State != "stop" {
		return errors.New("錯誤供應狀態")
	}
	return nil
}

type StockProductMaterial struct {
	*CommonStockProduct `json:",inline" bson:",inline"`
	Material            *SimpleProduct // 原物料
}

func (s *StockProductMaterial) GetUpdateFields() bson.D {
	updateFields := s.CommonStockProduct.GetUpdateFields()
	return append(updateFields, bson.E{Key: "material", Value: s.Material})
}

func (s *StockProductMaterial) Validate() error {
	if err := s.CommonStockProduct.Validate(); err != nil {
		return err
	}

	if s.Material == nil {
		return errors.New("請輸入原物料")
	} else {
		if s.Material.MaterialId == primitive.NilObjectID {
			return errors.New("請輸入原物料ID")
		}
		if s.Material.Unit == "" {
			return errors.New("請輸入配方單位")
		}
	}
	return nil
}

type StockProductFormulas struct {
	*CommonStockProduct `json:",inline" bson:",inline"`
	Formulas            []*Formula // 組合商品配方內容
}

func (s *StockProductFormulas) GetUpdateFields() bson.D {
	updateFields := s.CommonStockProduct.GetUpdateFields()
	return append(updateFields, bson.E{Key: "formulas", Value: s.Formulas})
}

func (s *StockProductFormulas) Validate() error {
	if err := s.CommonStockProduct.Validate(); err != nil {
		return err
	}
	if len(s.Formulas) == 0 {
		return errors.New("請輸入組合商品配方內容")
	} else {
		for u := range s.Formulas {
			if s.Formulas[u].Product.MaterialId == primitive.NilObjectID {
				return errors.New("請輸入組合商品配方ID")
			}
			if s.Formulas[u].Product.Unit == "" {
				return errors.New("請輸入組合商品配方單位")
			}
		}
	}
	return nil
}

type SimpleProduct struct {
	MaterialId primitive.ObjectID // 基本商品編號
	Unit       string             // 配方單位
	Remark     string             //備註
}

type Formula struct {
	Number  int
	Product *SimpleProduct
}

type Volume struct {
	Length int // 長
	Width  int // 寬
	Height int // 高
}
