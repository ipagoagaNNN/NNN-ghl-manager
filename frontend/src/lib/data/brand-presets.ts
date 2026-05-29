// Brand presets for the Custom Values bulk-fill feature.
// Extracted faithfully from the HTML prototype (BRAND_PRESETS_BASE, line ~6774).
//
// IMPORTANT: the keys are GHL custom-value NAMES and must match exactly what GHL
// returns for each location — including the source's "Testimoial" spelling. Do NOT
// "correct" these keys or the bulk-fill name-matching will silently miss.
//
// In the prototype these were overridable via localStorage (ghl_brand_presets_overrides_v1).
// That override layer is deferred — Phase 2c ships the base presets; per-user overrides
// will move server-side when auth lands (Phase 2/3).

export interface BrandPreset {
	key: string;
	label: string;
	color: string;
	values: Record<string, string>; // CV name → value
}

export const BRAND_PRESETS: BrandPreset[] = [
	{
		key: 'nnn',
		label: 'No Needle Needed',
		color: '#00b4d8',
		values: {
			Domain: 'No Needle Needed',
			'Location PIXEL ID': '540031443835405',
			'Privacy Policy': 'https://policy.noneedleneeded.com/policy',
			'Terms and Conditions': 'https://policy.noneedleneeded.com/terms',
			'FTB customer service email': 'hello@noneedleneeded.com',
		},
	},
	{
		key: 'ftb',
		label: 'First Touch Beauty',
		color: '#ff1d8d',
		values: {
			Domain: 'First Touch Beauty',
			'Location PIXEL ID': '1180664739949233',
			'Privacy Policy': 'https://policy.firstouchbeauty.com/policy',
			'Terms and Conditions': 'https://policy.firstouchbeauty.com/terms',
			'FTB customer service email': 'hello@firstouchbeauty.com',
		},
	},
	{
		key: 'advb',
		label: 'Adv Beauty Treatments',
		color: '#7b2ff7',
		values: {
			Domain:
				'© 2025 Advanced Beauty Treatments. <br> AdvancedBeautyTreatments.com is operated by No Needle Needed LLC.',
			'Location PIXEL ID': '819221296693583',
			'Privacy Policy': 'https://policy.advancedbeautytreatments.com/advb-policy',
			'Terms and Conditions': 'https://policy.advancedbeautytreatments.com/terms',
			'FTB customer service email': 'hello@noneedleneeded.com',
		},
	},
	{
		key: 'general',
		label: 'General CVs',
		color: '#00c97a',
		values: {
			'Before and After Face 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f41550b9a3263a132970.webp',
			'Before and After Face 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4158696a78b8d1ef956.webp',
			'Before and After Face 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f41538381eafa88ebe22.webp',
			'Before and After Face 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4152c135a8c8353de17.webp',
			'Before and After Face 5':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f415c56ad27908e1ca58.webp',
			'Before and After Face 6':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f415c56ad27908e1ca57.webp',
			'Before and After Face 7':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f41538381eafa88ebe23.webp',
			'After Face 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a438381eafa88edea6.webp',
			'After Face 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a5c56ad27908e1eaaa.webp',
			'After Face 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a5c56ad27908e1eaa8.png',
			'Before Face 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a5c56ad27908e1eaa9.webp',
			'Before Face 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a58696a78b8d1f193e.webp',
			'Before Face 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f4a52c135a8c8353fea1.webp',
			'Desktop Testimoial Thumbnail Link 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f6012c135a8c835449f3.webp',
			'Desktop Testimoial Thumbnail Link 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f6018696a78b8d1f6312.webp',
			'Desktop Testimoial Thumbnail Link 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f601c56ad27908e2360b.webp',
			'Desktop Testimoial Thumbnail Link 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f60150b9a3263a13957c.webp',
			'Desktop Testimoial Thumbnail Link 5':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f601c56ad27908e23609.webp',
			'Desktop Testimoial Video Link 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f60150b9a3263a139575.mp4',
			'Desktop Testimoial Video Link 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f6012c135a8c835449f4.mp4',
			'Desktop Testimoial Video Link 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f601c56ad27908e2360a.mp4',
			'Desktop Testimoial Video Link 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f6018696a78b8d1f6311.mp4',
			'Desktop Testimoial Video Link 5':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f6018696a78b8d1f630a.mp4',
			'General review 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5728696a78b8d1f43c8.webp',
			'General review 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5728696a78b8d1f43c7.webp',
			'General review 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f572c56ad27908e21510.webp',
			'General review 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5722c135a8c835428e1.webp',
			'Mobile Testimoial Video Link 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fdc.mp4',
			'Mobile Testimoial Video Link 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fe2.mp4',
			'Mobile Testimoial Video Link 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e550b9a3263a138ee9.mp4',
			'Mobile Testimoial Video Link 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e550b9a3263a138ef7.mp4',
			'Mobile Testimoial Video Link 5':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fdb.mp4',
			'Mobile Testimoial Thumbnail Link 1':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fe3.webp',
			'Mobile Testimoial Thumbnail Link 2':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fe1.webp',
			'Mobile Testimoial Thumbnail Link 3':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e550b9a3263a138ef4.webp',
			'Mobile Testimoial Thumbnail Link 4':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e5c56ad27908e22fe4.webp',
			'Mobile Testimoial Thumbnail Link 5':
				'https://assets.cdn.filesafe.space/6HyntUzpAjqcibyVmZpi/media/69e4f5e538381eafa88f2427.webp',
			'Txt General review 1':
				"Excellent experience from start to finish! The staff is incredibly skilled and caring, and my skin looks smoother and brighter than ever. I couldn't be happier with the results.",
			'Txt General review 2':
				'One of the best choices I did, amazing service with amazing people that are very professional and knowledgeable about anything related to your skin!',
			'Txt General review 3':
				"I'm so glad I stopped by a few weeks ago. I love the products and have really noticed a difference in my skin on such a short time!",
			'Txt General review 4':
				'An excellent experience with a highly professional team that is extremely knowledgeable about all things skin.',
			'Txt General review 5':
				'Wonderful products and a very professional, knowledgeable staff. My complexion has definitely improved and I am very pleased.',
		},
	},
];
