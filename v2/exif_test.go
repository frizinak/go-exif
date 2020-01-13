package exif

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"

	"encoding/binary"
	"io/ioutil"

	"github.com/dsoprea/go-logging"

	"github.com/dsoprea/go-exif/v2/common"
)

type innerVisitorCall func(fqIfdPath string, ifdIndex int, ite *IfdTagEntry) (err error)

type visitorWrapper struct {
	f innerVisitorCall
}

func (vw *visitorWrapper) Visit(fqIfdPath string, ifdIndex int, ite *IfdTagEntry) (err error) {
	return vw.f(fqIfdPath, ifdIndex, ite)
}

func TestVisit(t *testing.T) {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)

			t.Fatalf("Test failure.")
		}
	}()

	ti := NewTagIndex()

	// Open the file.

	testImageFilepath := getTestImageFilepath()

	f, err := os.Open(testImageFilepath)
	log.PanicIf(err)

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	log.PanicIf(err)

	// Search for the beginning of the EXIF information. The EXIF is near the
	// very beginning of our/most JPEGs, so this has a very low cost.

	foundAt := -1
	for i := 0; i < len(data); i++ {
		if _, err := ParseExifHeader(data[i:]); err == nil {
			foundAt = i
			break
		} else if log.Is(err, ErrNoExif) == false {
			log.Panic(err)
		}
	}

	if foundAt == -1 {
		log.Panicf("EXIF start not found")
	}

	// Run the parse.

	im := NewIfdMappingWithStandard()

	tags := make([]string, 0)

	visitor := func(fqIfdPath string, ifdIndex int, ite *IfdTagEntry) (err error) {
		defer func() {
			if state := recover(); state != nil {
				err = log.Wrap(state.(error))
				log.Panic(err)
			}
		}()

		tagId := ite.TagId()
		tagType := ite.TagType()

		ifdPath, err := im.StripPathPhraseIndices(fqIfdPath)
		log.PanicIf(err)

		it, err := ti.Get(ifdPath, tagId)
		if err != nil {
			if log.Is(err, ErrTagNotFound) {
				fmt.Printf("Unknown tag: [%s] (%04x)\n", ifdPath, tagId)
				return nil
			} else {
				log.Panic(err)
			}
		}

		valueString, err := ite.FormatFirst()
		log.PanicIf(err)

		description := fmt.Sprintf("IFD-PATH=[%s] ID=(0x%04x) NAME=[%s] COUNT=(%d) TYPE=[%s] VALUE=[%s]", ifdPath, tagId, it.Name, ite.UnitCount(), tagType.String(), valueString)
		tags = append(tags, description)

		return nil
	}

	vw := &visitorWrapper{
		f: visitor,
	}

	_, err = Visit(exifcommon.IfdStandard, im, ti, data[foundAt:], vw)
	log.PanicIf(err)

	expected := []string{
		"IFD-PATH=[IFD] ID=(0x010f) NAME=[Make] COUNT=(6) TYPE=[ASCII] VALUE=[Canon]",
		"IFD-PATH=[IFD] ID=(0x0110) NAME=[Model] COUNT=(22) TYPE=[ASCII] VALUE=[Canon EOS 5D Mark III]",
		"IFD-PATH=[IFD] ID=(0x0112) NAME=[Orientation] COUNT=(1) TYPE=[SHORT] VALUE=[1]",
		"IFD-PATH=[IFD] ID=(0x011a) NAME=[XResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[72/1]",
		"IFD-PATH=[IFD] ID=(0x011b) NAME=[YResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[72/1]",
		"IFD-PATH=[IFD] ID=(0x0128) NAME=[ResolutionUnit] COUNT=(1) TYPE=[SHORT] VALUE=[2]",
		"IFD-PATH=[IFD] ID=(0x0132) NAME=[DateTime] COUNT=(20) TYPE=[ASCII] VALUE=[2017:12:02 08:18:50]",
		"IFD-PATH=[IFD] ID=(0x013b) NAME=[Artist] COUNT=(1) TYPE=[ASCII] VALUE=[]",
		"IFD-PATH=[IFD] ID=(0x0213) NAME=[YCbCrPositioning] COUNT=(1) TYPE=[SHORT] VALUE=[2]",
		"IFD-PATH=[IFD] ID=(0x8298) NAME=[Copyright] COUNT=(1) TYPE=[ASCII] VALUE=[]",
		"IFD-PATH=[IFD] ID=(0x8769) NAME=[ExifTag] COUNT=(1) TYPE=[LONG] VALUE=[360]",
		"IFD-PATH=[IFD/Exif] ID=(0x829a) NAME=[ExposureTime] COUNT=(1) TYPE=[RATIONAL] VALUE=[1/640]",
		"IFD-PATH=[IFD/Exif] ID=(0x829d) NAME=[FNumber] COUNT=(1) TYPE=[RATIONAL] VALUE=[4/1]",
		"IFD-PATH=[IFD/Exif] ID=(0x8822) NAME=[ExposureProgram] COUNT=(1) TYPE=[SHORT] VALUE=[4]",
		"IFD-PATH=[IFD/Exif] ID=(0x8827) NAME=[ISOSpeedRatings] COUNT=(1) TYPE=[SHORT] VALUE=[1600]",
		"IFD-PATH=[IFD/Exif] ID=(0x8830) NAME=[SensitivityType] COUNT=(1) TYPE=[SHORT] VALUE=[2]",
		"IFD-PATH=[IFD/Exif] ID=(0x8832) NAME=[RecommendedExposureIndex] COUNT=(1) TYPE=[LONG] VALUE=[1600]",
		"IFD-PATH=[IFD/Exif] ID=(0x9000) NAME=[ExifVersion] COUNT=(4) TYPE=[UNDEFINED] VALUE=[0230]",
		"IFD-PATH=[IFD/Exif] ID=(0x9003) NAME=[DateTimeOriginal] COUNT=(20) TYPE=[ASCII] VALUE=[2017:12:02 08:18:50]",
		"IFD-PATH=[IFD/Exif] ID=(0x9004) NAME=[DateTimeDigitized] COUNT=(20) TYPE=[ASCII] VALUE=[2017:12:02 08:18:50]",
		"IFD-PATH=[IFD/Exif] ID=(0x9101) NAME=[ComponentsConfiguration] COUNT=(4) TYPE=[UNDEFINED] VALUE=[Exif9101ComponentsConfiguration<ID=[YCBCR] BYTES=[1 2 3 0]>]",
		"IFD-PATH=[IFD/Exif] ID=(0x9201) NAME=[ShutterSpeedValue] COUNT=(1) TYPE=[SRATIONAL] VALUE=[614400/65536]",
		"IFD-PATH=[IFD/Exif] ID=(0x9202) NAME=[ApertureValue] COUNT=(1) TYPE=[RATIONAL] VALUE=[262144/65536]",
		"IFD-PATH=[IFD/Exif] ID=(0x9204) NAME=[ExposureBiasValue] COUNT=(1) TYPE=[SRATIONAL] VALUE=[0/1]",
		"IFD-PATH=[IFD/Exif] ID=(0x9207) NAME=[MeteringMode] COUNT=(1) TYPE=[SHORT] VALUE=[5]",
		"IFD-PATH=[IFD/Exif] ID=(0x9209) NAME=[Flash] COUNT=(1) TYPE=[SHORT] VALUE=[16]",
		"IFD-PATH=[IFD/Exif] ID=(0x920a) NAME=[FocalLength] COUNT=(1) TYPE=[RATIONAL] VALUE=[16/1]",
		"IFD-PATH=[IFD/Exif] ID=(0x927c) NAME=[MakerNote] COUNT=(8152) TYPE=[UNDEFINED] VALUE=[MakerNote<TYPE-ID=[28 00 01 00 03 00 31 00 00 00 74 05 00 00 02 00 03 00 04 00] LEN=(8152) SHA1=[d4154aa7df5474efe7ab38de2595919b9b4cc29f]>]",
		"IFD-PATH=[IFD/Exif] ID=(0x9286) NAME=[UserComment] COUNT=(264) TYPE=[UNDEFINED] VALUE=[UserComment<SIZE=(256) ENCODING=[UNDEFINED] V=[0 0 0 0 0 0 0 0]... LEN=(256)>]",
		"IFD-PATH=[IFD/Exif] ID=(0x9290) NAME=[SubSecTime] COUNT=(3) TYPE=[ASCII] VALUE=[00]",
		"IFD-PATH=[IFD/Exif] ID=(0x9291) NAME=[SubSecTimeOriginal] COUNT=(3) TYPE=[ASCII] VALUE=[00]",
		"IFD-PATH=[IFD/Exif] ID=(0x9292) NAME=[SubSecTimeDigitized] COUNT=(3) TYPE=[ASCII] VALUE=[00]",
		"IFD-PATH=[IFD/Exif] ID=(0xa000) NAME=[FlashpixVersion] COUNT=(4) TYPE=[UNDEFINED] VALUE=[0100]",
		"IFD-PATH=[IFD/Exif] ID=(0xa001) NAME=[ColorSpace] COUNT=(1) TYPE=[SHORT] VALUE=[1]",
		"IFD-PATH=[IFD/Exif] ID=(0xa002) NAME=[PixelXDimension] COUNT=(1) TYPE=[SHORT] VALUE=[3840]",
		"IFD-PATH=[IFD/Exif] ID=(0xa003) NAME=[PixelYDimension] COUNT=(1) TYPE=[SHORT] VALUE=[2560]",
		"IFD-PATH=[IFD/Exif] ID=(0xa005) NAME=[InteroperabilityTag] COUNT=(1) TYPE=[LONG] VALUE=[9326]",
		"IFD-PATH=[IFD/Exif/Iop] ID=(0x0001) NAME=[InteroperabilityIndex] COUNT=(4) TYPE=[ASCII] VALUE=[R98]",
		"IFD-PATH=[IFD/Exif/Iop] ID=(0x0002) NAME=[InteroperabilityVersion] COUNT=(4) TYPE=[UNDEFINED] VALUE=[0100]",
		"IFD-PATH=[IFD/Exif] ID=(0xa20e) NAME=[FocalPlaneXResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[3840000/1461]",
		"IFD-PATH=[IFD/Exif] ID=(0xa20f) NAME=[FocalPlaneYResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[2560000/972]",
		"IFD-PATH=[IFD/Exif] ID=(0xa210) NAME=[FocalPlaneResolutionUnit] COUNT=(1) TYPE=[SHORT] VALUE=[2]",
		"IFD-PATH=[IFD/Exif] ID=(0xa401) NAME=[CustomRendered] COUNT=(1) TYPE=[SHORT] VALUE=[0]",
		"IFD-PATH=[IFD/Exif] ID=(0xa402) NAME=[ExposureMode] COUNT=(1) TYPE=[SHORT] VALUE=[0]",
		"IFD-PATH=[IFD/Exif] ID=(0xa403) NAME=[WhiteBalance] COUNT=(1) TYPE=[SHORT] VALUE=[0]",
		"IFD-PATH=[IFD/Exif] ID=(0xa406) NAME=[SceneCaptureType] COUNT=(1) TYPE=[SHORT] VALUE=[0]",
		"IFD-PATH=[IFD/Exif] ID=(0xa430) NAME=[CameraOwnerName] COUNT=(1) TYPE=[ASCII] VALUE=[]",
		"IFD-PATH=[IFD/Exif] ID=(0xa431) NAME=[BodySerialNumber] COUNT=(13) TYPE=[ASCII] VALUE=[063024020097]",
		"IFD-PATH=[IFD/Exif] ID=(0xa432) NAME=[LensSpecification] COUNT=(4) TYPE=[RATIONAL] VALUE=[16/1...]",
		"IFD-PATH=[IFD/Exif] ID=(0xa434) NAME=[LensModel] COUNT=(22) TYPE=[ASCII] VALUE=[EF16-35mm f/4L IS USM]",
		"IFD-PATH=[IFD/Exif] ID=(0xa435) NAME=[LensSerialNumber] COUNT=(11) TYPE=[ASCII] VALUE=[2400001068]",
		"IFD-PATH=[IFD] ID=(0x8825) NAME=[GPSTag] COUNT=(1) TYPE=[LONG] VALUE=[9554]",
		"IFD-PATH=[IFD/GPSInfo] ID=(0x0000) NAME=[GPSVersionID] COUNT=(4) TYPE=[BYTE] VALUE=[02 03 00 00]",
		"IFD-PATH=[IFD] ID=(0x0103) NAME=[Compression] COUNT=(1) TYPE=[SHORT] VALUE=[6]",
		"IFD-PATH=[IFD] ID=(0x011a) NAME=[XResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[72/1]",
		"IFD-PATH=[IFD] ID=(0x011b) NAME=[YResolution] COUNT=(1) TYPE=[RATIONAL] VALUE=[72/1]",
		"IFD-PATH=[IFD] ID=(0x0128) NAME=[ResolutionUnit] COUNT=(1) TYPE=[SHORT] VALUE=[2]",
	}

	if reflect.DeepEqual(tags, expected) == false {
		fmt.Printf("\n")
		fmt.Printf("ACTUAL:\n")
		fmt.Printf("\n")

		for _, line := range tags {
			fmt.Println(line)
		}

		fmt.Printf("\n")
		fmt.Printf("EXPECTED:\n")
		fmt.Printf("\n")

		for _, line := range expected {
			fmt.Println(line)
		}

		fmt.Printf("\n")

		t.Fatalf("tags not correct.")
	}
}

func TestSearchFileAndExtractExif(t *testing.T) {
	testImageFilepath := getTestImageFilepath()

	// Returns a slice starting with the EXIF data and going to the end of the
	// image.
	rawExif, err := SearchFileAndExtractExif(testImageFilepath)
	log.PanicIf(err)

	testExifData := getTestExifData()

	if bytes.Compare(rawExif[:len(testExifData)], testExifData) != 0 {
		t.Fatalf("found EXIF data not correct")
	}
}

func TestSearchAndExtractExif(t *testing.T) {
	testImageFilepath := getTestImageFilepath()

	imageData, err := ioutil.ReadFile(testImageFilepath)
	log.PanicIf(err)

	rawExif, err := SearchAndExtractExif(imageData)
	log.PanicIf(err)

	testExifData := getTestExifData()

	if bytes.Compare(rawExif[:len(testExifData)], testExifData) != 0 {
		t.Fatalf("found EXIF data not correct")
	}
}

func TestCollect(t *testing.T) {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)

			t.Fatalf("Test failure.")
		}
	}()

	testImageFilepath := getTestImageFilepath()

	rawExif, err := SearchFileAndExtractExif(testImageFilepath)
	log.PanicIf(err)

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	rootIfd := index.RootIfd
	ifds := index.Ifds
	tree := index.Tree
	lookup := index.Lookup

	if rootIfd.Offset != uint32(0x0008) {
		t.Fatalf("Root-IFD not correct: (0x%04d).", rootIfd.Offset)
	} else if rootIfd.Id != 0 {
		t.Fatalf("Root-IFD does not have the right ID: (%d)", rootIfd.Id)
	} else if tree[0] != rootIfd {
		t.Fatalf("Root-IFD is not indexed properly.")
	} else if len(ifds) != 5 {
		t.Fatalf("The IFD list is not the right size: (%d)", len(ifds))
	} else if len(tree) != 5 {
		t.Fatalf("The IFD tree is not the right size: (%d)", len(tree))
	} else if len(lookup) != 4 {
		t.Fatalf("The IFD lookup is not the right size: (%d)", len(lookup))
	}

	if rootIfd.NextIfdOffset != 0x2c54 {
		t.Fatalf("Root IFD does not continue correctly: (0x%04x)", rootIfd.NextIfdOffset)
	} else if rootIfd.NextIfd.Offset != rootIfd.NextIfdOffset {
		t.Fatalf("Root IFD neighbor object does not have the right offset: (0x%04x != 0x%04x)", rootIfd.NextIfd.Offset, rootIfd.NextIfdOffset)
	} else if rootIfd.NextIfd.NextIfdOffset != 0 {
		t.Fatalf("Root IFD chain not terminated correctly (1).")
	} else if rootIfd.NextIfd.NextIfd != nil {
		t.Fatalf("Root IFD chain not terminated correctly (2).")
	}

	if rootIfd.IfdPath != exifcommon.IfdPathStandard {
		t.Fatalf("Root IFD is not labeled correctly: [%s]", rootIfd.IfdPath)
	} else if rootIfd.NextIfd.IfdPath != exifcommon.IfdPathStandard {
		t.Fatalf("Root IFD sibling is not labeled correctly: [%s]", rootIfd.IfdPath)
	} else if rootIfd.Children[0].IfdPath != exifcommon.IfdPathStandardExif {
		t.Fatalf("Root IFD child (0) is not labeled correctly: [%s]", rootIfd.Children[0].IfdPath)
	} else if rootIfd.Children[1].IfdPath != exifcommon.IfdPathStandardGps {
		t.Fatalf("Root IFD child (1) is not labeled correctly: [%s]", rootIfd.Children[1].IfdPath)
	} else if rootIfd.Children[0].Children[0].IfdPath != exifcommon.IfdPathStandardExifIop {
		t.Fatalf("Exif IFD child is not an IOP IFD: [%s]", rootIfd.Children[0].Children[0].IfdPath)
	}

	if lookup[exifcommon.IfdPathStandard][0].IfdPath != exifcommon.IfdPathStandard {
		t.Fatalf("Lookup for standard IFD not correct.")
	} else if lookup[exifcommon.IfdPathStandard][1].IfdPath != exifcommon.IfdPathStandard {
		t.Fatalf("Lookup for standard IFD not correct.")
	}

	if lookup[exifcommon.IfdPathStandardExif][0].IfdPath != exifcommon.IfdPathStandardExif {
		t.Fatalf("Lookup for EXIF IFD not correct.")
	}

	if lookup[exifcommon.IfdPathStandardGps][0].IfdPath != exifcommon.IfdPathStandardGps {
		t.Fatalf("Lookup for GPS IFD not correct.")
	}

	if lookup[exifcommon.IfdPathStandardExifIop][0].IfdPath != exifcommon.IfdPathStandardExifIop {
		t.Fatalf("Lookup for IOP IFD not correct.")
	}

	foundExif := 0
	foundGps := 0
	for _, ite := range lookup[exifcommon.IfdPathStandard][0].Entries {
		if ite.ChildIfdPath() == exifcommon.IfdPathStandardExif {
			foundExif++

			if ite.TagId() != exifcommon.IfdExifId {
				t.Fatalf("EXIF IFD tag-ID mismatch: (0x%04x) != (0x%04x)", ite.TagId(), exifcommon.IfdExifId)
			}
		}

		if ite.ChildIfdPath() == exifcommon.IfdPathStandardGps {
			foundGps++

			if ite.TagId() != exifcommon.IfdGpsId {
				t.Fatalf("GPS IFD tag-ID mismatch: (0x%04x) != (0x%04x)", ite.TagId(), exifcommon.IfdGpsId)
			}
		}
	}

	if foundExif != 1 {
		t.Fatalf("Exactly one EXIF IFD tag wasn't found: (%d)", foundExif)
	} else if foundGps != 1 {
		t.Fatalf("Exactly one GPS IFD tag wasn't found: (%d)", foundGps)
	}

	foundIop := 0
	for _, ite := range lookup[exifcommon.IfdPathStandardExif][0].Entries {
		if ite.ChildIfdPath() == exifcommon.IfdPathStandardExifIop {
			foundIop++

			if ite.TagId() != exifcommon.IfdIopId {
				t.Fatalf("IOP IFD tag-ID mismatch: (0x%04x) != (0x%04x)", ite.TagId(), exifcommon.IfdIopId)
			}
		}
	}

	if foundIop != 1 {
		t.Fatalf("Exactly one IOP IFD tag wasn't found: (%d)", foundIop)
	}
}

func TestParseExifHeader(t *testing.T) {
	testExifData := getTestExifData()

	eh, err := ParseExifHeader(testExifData)
	log.PanicIf(err)

	if eh.ByteOrder != binary.LittleEndian {
		t.Fatalf("Byte-order of EXIF header not correct.")
	} else if eh.FirstIfdOffset != 0x8 {
		t.Fatalf("First IFD offset not correct.")
	}
}

func TestExif_BuildAndParseExifHeader(t *testing.T) {
	headerBytes, err := BuildExifHeader(exifcommon.TestDefaultByteOrder, 0x11223344)
	log.PanicIf(err)

	eh, err := ParseExifHeader(headerBytes)
	log.PanicIf(err)

	if eh.ByteOrder != exifcommon.TestDefaultByteOrder {
		t.Fatalf("Byte-order of EXIF header not correct.")
	} else if eh.FirstIfdOffset != 0x11223344 {
		t.Fatalf("First IFD offset not correct.")
	}
}

func ExampleBuildExifHeader() {
	headerBytes, err := BuildExifHeader(exifcommon.TestDefaultByteOrder, 0x11223344)
	log.PanicIf(err)

	eh, err := ParseExifHeader(headerBytes)
	log.PanicIf(err)

	fmt.Printf("%v\n", eh)
	// Output: ExifHeader<BYTE-ORDER=[BigEndian] FIRST-IFD-OFFSET=(0x11223344)>
}
