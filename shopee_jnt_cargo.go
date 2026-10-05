package san_receipt_readers

// shopeeJNTCargoLayout is a Shopee label carried by J&T Cargo (service CARGO), printed through PDFium.
// Measured on one sample (shopee_cargo_01.pdf).
//
// ⚠ ITS NUMBER HAS SICEPAT'S SHAPE: twelve digits. The text never names J&T, so the layout needs J&T
// Cargo's logo (courierLogos) — without it, a Shopee label with twelve digits stays unknown rather than
// being read as either courier's.
var shopeeJNTCargoLayout = layout{
	name: "shopee_jnt_cargo",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) &&
			logoCourier(v) == courierJNTCargo &&
			readShopeeTracking(v.lines, jntCargoTracking) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readShopeeTracking(v.lines, jntCargoTracking),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}
