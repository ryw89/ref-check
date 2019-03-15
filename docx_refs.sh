#!/bin/bash

# Wrapper around docx_refs.py to prevent overwriting of .docx file.
if [ "$#" -ne 4 ]
then
	echo "docx_ref.sh: Arguments missing."
	echo "Usage: ./docx_refs.sh file.docx 'Name of references section' 'Name of section after references'"
	echo "Alternatively, you may specify 'None' for the section after references."
	exit
fi

docx_file    = "$1"
ref_sec      = "$2"
afterref_sec = "$3"

cp -i "$docx_file" /tmp

./docx_refs.py /tmp/"$docx_file" "$ref_sec" "$afterref_sec"
cp -i /tmp/"$docx_file" ./docx_refs_reponse.html

echo "Wrote docx_ref_response.html."
