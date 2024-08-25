package mysql

import (
	"context"
	"gorm.io/gorm"
	"xzdp/biz/model/voucher"
)

func QueryVoucherByID(ctx context.Context, id int64) ([]*voucher.Voucher, error) {
	var voucherList []*voucher.Voucher

	err = DB.WithContext(ctx).Where("shop_id = ?", id).Find(&voucherList).Error
	if err != nil {
		return nil, err
	}

	return voucherList, nil
}

func QuerySeckillVoucherByID(ctx context.Context, tx *gorm.DB, id int64) (*voucher.SeckillVoucher, error) {
	var seckillVoucher *voucher.SeckillVoucher

	err = tx.WithContext(ctx).Where("voucher_id = ?", id).Find(&seckillVoucher).Error
	if err != nil {
		return nil, err
	}

	return seckillVoucher, nil
}

func SubSeckillStockVoucherById(ctx context.Context, tx *gorm.DB, id int64) error {
	err = tx.WithContext(ctx).Model(&voucher.SeckillVoucher{}).
		Where("voucher_id = ? AND stock > 0", id).
		UpdateColumn("stock", gorm.Expr("stock - 1")).Error
	if err != nil {
		return err
	}
	return nil
}

func CountSeckillOrderByUserId(ctx context.Context, tx *gorm.DB, userId int64) (count int64, err error) {
	err = tx.WithContext(ctx).Model(&voucher.VoucherOrder{}).Where("user_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddVoucherOrder(ctx context.Context, tx *gorm.DB, voucher *voucher.VoucherOrder) error {
	err = tx.WithContext(ctx).Create(voucher).Error

	if err != nil {
		return err
	}
	return nil
}
