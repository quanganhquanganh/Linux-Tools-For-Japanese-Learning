# Usage: jisho <word>
# Example: jisho おはよう

jisho() {
        # Percent encode the japanese text
        WORD=$(echo $1 | perl -MURI::Escape -ne 'print uri_escape($_)')
        RES=$(curl -s "https://jisho.org/api/v1/search/words?keyword=$WORD")
        jq -r '.data[] | .japanese[] | .word + " (" + .reading + ")"' <<< $RES
        jq -r '.data[] | .senses[] | .english_definitions[]' <<< $RES
        # Get audio from japanesepod101 using first kanji and reading
        jq -r '.data[] | .japanese[] | .word' <<< $RES | while read -r kanji; do
                jq -r '.data[] | .japanese[] | .reading' <<< $RES | while read -r reading; do
                        curl -s "https://assets.languagepod101.com/dictionary/japanese/audiomp3.php?kanji=$kanji&kana=$reading" | mpv -
                        break
                done
                break
        done
}
