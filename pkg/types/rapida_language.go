// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"strings"
)

// Language holds language name and ISO codes
type Language struct {
	Name     string
	ISO639_1 string
	ISO639_2 string
}

var UNKNOWN_LANGUAGE = Language{
	Name:     "Unknown",
	ISO639_1: "unknown",
	ISO639_2: "unknown",
}

var languages = map[string]Language{
	"unknown": UNKNOWN_LANGUAGE,

	// 🌍 International Languages
	"en": {"English", "en", "eng"},
	"fr": {"French", "fr", "fra"},
	"es": {"Spanish", "es", "spa"},
	"de": {"German", "de", "deu"},
	"it": {"Italian", "it", "ita"},
	"pt": {"Portuguese", "pt", "por"},
	"ru": {"Russian", "ru", "rus"},
	"zh": {"Chinese", "zh", "zho"},
	"ja": {"Japanese", "ja", "jpn"},
	"ko": {"Korean", "ko", "kor"},
	"ar": {"Arabic", "ar", "ara"},
	"tr": {"Turkish", "tr", "tur"},
	"nl": {"Dutch", "nl", "nld"},
	"pl": {"Polish", "pl", "pol"},
	"sv": {"Swedish", "sv", "swe"},
	"no": {"Norwegian", "no", "nor"},
	"da": {"Danish", "da", "dan"},
	"fi": {"Finnish", "fi", "fin"},
	"he": {"Hebrew", "he", "heb"},
	"el": {"Greek", "el", "ell"},

	// 🇮🇳 Indian Languages
	"hi": {"Hindi", "hi", "hin"},
	"bn": {"Bengali", "bn", "ben"},
	"te": {"Telugu", "te", "tel"},
	"mr": {"Marathi", "mr", "mar"},
	"ta": {"Tamil", "ta", "tam"},
	"ur": {"Urdu", "ur", "urd"},
	"gu": {"Gujarati", "gu", "guj"},
	"kn": {"Kannada", "kn", "kan"},
	"ml": {"Malayalam", "ml", "mal"},
	"or": {"Odia", "or", "ori"},
	"pa": {"Punjabi", "pa", "pan"},
	"as": {"Assamese", "as", "asm"},
	"sa": {"Sanskrit", "sa", "san"},
	"sd": {"Sindhi", "sd", "snd"},
	"ks": {"Kashmiri", "ks", "kas"},
}

// LookupLanguage returns a language by key without fallback.
func LookupLanguage(name string) Language {
	lang, ok := languages[strings.ToLower(name)]
	if !ok {
		return UNKNOWN_LANGUAGE
	}
	return lang
}
