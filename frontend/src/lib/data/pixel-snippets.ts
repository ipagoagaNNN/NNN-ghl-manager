// Meta Pixel snippets for assisted-manual pixel fixing (Phase 2e-2).
//
// The public GHL API v2 cannot write funnel tracking code, so the tool hands the
// operator the exact snippet to paste into GHL → Funnels → Settings → Tracking
// Code (Head). Brand list mirrors the backend's expectedPixels (handlers/pixels.go);
// the snippet is template-generated so the pixel ID lives in exactly one place.
//
// Source: prototype PIXEL_SCRIPTS (ghl-manager-final.2026-04-24-123140.html L6118).

export interface BrandPixel {
	domainKey: string;
	brand: string;
	pixelId: string;
}

export const brandPixels: BrandPixel[] = [
	{ domainKey: 'firstouchbeauty.com', brand: 'First Touch Beauty', pixelId: '1180664739949233' },
	{ domainKey: 'noneedleneeded.com', brand: 'No Needle Needed', pixelId: '540031443835405' },
	{
		domainKey: 'advancedbeautytreatments.com',
		brand: 'Adv Beauty Treatments',
		pixelId: '819221296693583'
	}
];

// pixelSnippet returns the standard Meta Pixel install code for a pixel ID.
// Matches the prototype's PIXEL_SCRIPTS code verbatim (only the ID varies).
export function pixelSnippet(pixelId: string): string {
	return `<!-- Meta Pixel Code -->
<script>
  !function(f,b,e,v,n,t,s)
  {if(f.fbq)return;n=f.fbq=function(){n.callMethod?
  n.callMethod.apply(n,arguments):n.queue.push(arguments)};
  if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
  n.queue=[];t=b.createElement(e);t.async=!0;
  t.src=v;s=b.getElementsByTagName(e)[0];
  s.parentNode.insertBefore(t,s)}(window, document,'script',
  'https://connect.facebook.net/en_US/fbevents.js');
  fbq('init', '${pixelId}');
  fbq('track', 'PageView');
<\/script>
<noscript>
  <img height="1" width="1" style="display:none"
    src="https://www.facebook.com/tr?id=${pixelId}&ev=PageView&noscript=1"/>
</noscript>
<!-- End Meta Pixel Code -->`;
}

// brandForDomain maps a domain to its configured brand pixel (substring match,
// mirroring the backend's expectedPixelForDomain).
export function brandForDomain(domain: string): BrandPixel | undefined {
	const d = (domain || '').toLowerCase();
	return brandPixels.find((b) => d.includes(b.domainKey));
}

// ghlFunnelsURL deep-links to a location's funnels area in the GHL app so the
// operator can paste the snippet. Best-effort path (GHL app v2 layout).
export function ghlFunnelsURL(locationId: string): string {
	return `https://app.gohighlevel.com/v2/location/${encodeURIComponent(locationId)}/funnels-websites/funnels`;
}
