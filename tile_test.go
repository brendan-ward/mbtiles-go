package mbtiles

import (
	"encoding/hex"
	"testing"
)

func Test_DetectTileFormat(t *testing.T) {
	tests := []struct {
		data   string
		format TileFormat
	}{
		{
			// PNG, first 20 bytes of tile 0/0/0 in geography-class-png.mbtiles
			data: "89504e470d0a1a0a0000000d4948445200000100", format: PNG,
		},
		{
			// JPG: first 20 bytes of tile 0/0/0 in geography-class-jpg.mbtiles
			data: "ffd8ffe000104a46494600010100000100010000", format: JPG,
		},
		{
			// Lossy webp
			// first 28 bytes of https://www.gstatic.com/webp/gallery/1.webp
			data: "52494646e22800005745425056503820d628000092b3009d012a4001", format: WEBP,
		},
		{
			// Lossless webp
			// first 24 bytes of https://www.gstatic.com/webp/gallery3/1_webp_ll.webp
			data: "52494646a43f0100574542505650384c983f01002f8f014b", format: WEBP,
		},
		{
			// PBF, first 10 bytes of tile 0/0/0 in world_cities.mbtiles
			// is detected as a GZIP and handled as a PBF later
			data: "1f8b0800000000000203", format: GZIP,
		},
	}

	for _, tc := range tests {
		data, err := hex.DecodeString(tc.data)
		if err != nil {
			t.Error("Error decoding hex image data", err)
		}

		format, err := detectTileFormat(data)
		if err != nil {
			t.Error("Error detecting tile format:", err)
		}
		if format != tc.format {
			t.Error("Tile format", format, "does not match expected value", tc.format)
		}
	}
}

// Since not all encodings are present in test mbtiles files, a selection of sample
// images are used for this test
func Test_DetectTilesize(t *testing.T) {
	tests := []struct {
		format   TileFormat
		data     string
		tilesize uint32
	}{
		{
			// PNG, first 20 bytes of tile 0/0/0 in geography-class-png.mbtiles
			format: PNG, data: "89504e470d0a1a0a0000000d4948445200000100", tilesize: 256,
		},
		{
			// JPG, all bytes of https://www.w3.org/People/mimasa/test/imgformat/img/w3c_home.jpg
			format: JPG, data: "ffd8ffe000104a46494600010100000100010000ffdb00430006040506050406060506070706080a100a0a09090a140e0f0c1017141818171416161a1d251f1a1b231c1616202c20232627292a29191f2d302d283025282928ffdb0043010707070a080a130a0a13281a161a2828282828282828282828282828282828282828282828282828282828282828282828282828282828282828282828282828ffc00011080030004803012200021101031101ffc4001f0000010501010101010100000000000000000102030405060708090a0bffc400b5100002010303020403050504040000017d01020300041105122131410613516107227114328191a1082342b1c11552d1f02433627282090a161718191a25262728292a3435363738393a434445464748494a535455565758595a636465666768696a737475767778797a838485868788898a92939495969798999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d9dae1e2e3e4e5e6e7e8e9eaf1f2f3f4f5f6f7f8f9faffc4001f0100030101010101010101010000000000000102030405060708090a0bffc400b51100020102040403040705040400010277000102031104052131061241510761711322328108144291a1b1c109233352f0156272d10a162434e125f11718191a262728292a35363738393a434445464748494a535455565758595a636465666768696a737475767778797a82838485868788898a92939495969798999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d9dae2e3e4e5e6e7e8e9eaf2f3f4f5f6f7f8f9faffda000c03010002110311003f00f7ff00f8571e08ff00a137c37ff82b83ff0089a3fe15c7823fe84df0dffe0ae0ff00e26bcc7f68db8d5f42d6b42d6b4bd42f2085d4c4f147332c7bd1b702541c1c8623e8b5ebf3b2789bc1eef6333c2ba9596e86546dac9bd32ac08e84641fc2ba6787e5a70a97d25f818c6af34a50b6a8ceff008571e08ffa137c37ff0082b83ff89a3fe15c7823fe84df0dff00e0ae0ffe26bcd3f66bf12df5dddeb9a2eaf797171711edb88bcf94bb2e0ec71924f19d9fad745fb45f8825d1bc0c96d693bc3757f70b1868d8ab045f998823dc28ff008155cb0728e2161efaf7263888ba5ed4ea7fe15c7823fe84df0dff00e0ae0ffe268ff8571e08ff00a137c37ff82b83ff0089af06b5f0278e57c249e2097c4e963a7b5b7dacacd7f3abaa119190aa4648c719cf2075abdfb3ec1e20f1078acea377ab6a32699a70264592e1d96476042a609c1ee4fd07ad6f3cbe3184a6aa26a2651c5b7251706ae7b5ff00c2b8f047fd09be1bff00c15c1ffc4d1ff0ae3c11ff00426f86ff00f05707ff00135e03f147c63e3ef885f13350f087c319ef2d6cb486f2ee67b59fc82d229c3b3ca082aa1b2a141e769383d02e9de39f89fa7eb3e1df87be24b295b5f9356b69d6fcb0c5c59236f752cbc30f93961ced0c08cf5f30ed3d22eecfc076fe227d33fe15ff0086996346959fec5079822569159f6797d01864efd97f8995495e8171e13d327bf375221219fcc688aa105be627e62bbc025df2a1803bdf23e66c9401cefc78d17fb67e1b6a451374d6256f13db66771ffbe0bd727f0d3c67f63f811a8ddb49fe95a324b6e84f5dcdcc5f865c2ffc06bda2eade3bab59ade750f0ca863753dd48c11fad7c4ba9bea1a03ebbe160c4c6f78a93281cbb44ce171f5dd9f7e2bd8c04162693a32e8d3f9753cfc549d19aa8baa6bfc8dbf845a84fe1cf1ff87afae51a3b4bf630ef3c2ba3b18f39f40e33ff0001aedfe37c8fe2bf8b3a27862ddc9483cb85f1fc0d290cedf826c3f854ff001abc1dfd8bf0d7c2935baed9f490b6f3327ab8dccdff007f14ff00df754fe034771e2bf8a1aaf89b5201a4811a6240e16493e5503d82efc7d0576caa4669e35744d7cefa1cca128ffb33ead3ff003343f694f159812d3c25a7e63855166b9dbc0207fab4fa0c67fef9f4aee3e026a1e1e97c15058681313736e37dec72aed90cadd588ee38c0209e001d45777a968ba5ea6b22ea3a759dd09176b79d0ab923d32457ce1e1a813c0ffb41b69b6126cb033180ab3f1e5491870a4ffb24a9e7fbbcd705270c4619d18e8e2afea754d4a8d6551ea9e9e87a3fecd5a1ae9be009b53997fe263acdfdcdddd3b0f98912b2007e81738ec58d7a7cd636b35edb5e4d6f13dd5b0758656505a30f8dc14f6ce067e950689a5c1a3d87d8ed0621134b301e9e648d2103d817357c9c75af24ef0a28a2800af08f17f82fed7fb41e8d308f36778ab7d2923e5cc23e61f8ed8ffefbaf4dff00858fe08ffa1cbc37ff0083483ff8ba8dbe207809ae1276f17785ccf1ab2248752b7dcaac41600eec8076ae477c0f4adf0f5e541b71ea9a32ab4954493e8ee5ef883a2ffc243e0bd634c0bbe49eddbca1ff004d17e64ffc780af33f833e1dd4edfe0f6a571a2cff0064d6b546796da52a0e027caabc8c0c956e7b6ecd7a17fc2c7f047fd0e5e1bffc1a41ff00c5d476ff00103c056d0a436de2ef0bc5120c2a47a95baa8fa00d554f13285274d774feefe90a5454a7cfe563c87c1bf1a6f7c37a7dc693e31b0bebcbfb6770b2b3e25c924ec93773c1efe9818e39a5f0a342bbf885e3fd4bc4dad5b674d26532e47c8eeea5044a7bed56cfb607ad7adea7e26f85daacc26d535bf055ecc06d0f73776b2301e99626afdaf8ff00c03696e905af8b7c2d04118c2471ea56eaaa3d000d815d52c6d35197b285a52dff00e018c70d3bae795d2d8e093e2e7fc2babe7f0dfc508af44d013f62d66287cc8efe01f75d80e4498c0600119e78c8cf2f77f11b57f8d1e2fd33c3fe00b6bbb1f0ed95dc379a96a53aed6658dc32ae01200caf0b9cb103a006bd7357f18fc36d66d7ecdac788bc1f7f6d9cf95757d6d2a67d70cc453b4af19fc38d22d16d749f127842c6d57910db5f5b4483fe02ac05798761db515caffc2c7f047fd0e5e1bffc1a41ff00c5d1401fffd9", tilesize: 72,
		},
		{
			// Lossy webp
			// first 28 bytes of https://www.gstatic.com/webp/gallery/1.webp
			format: WEBP, data: "52494646e22800005745425056503820d628000092b3009d012a4001", tilesize: 320,
		},
		{
			// Lossless webp
			// first 24 bytes of https://www.gstatic.com/webp/gallery3/1_webp_ll.webp
			format: WEBP, data: "52494646a43f0100574542505650384c983f01002f8f014b", tilesize: 400,
		},
		{
			// PBF, first 10 bytes of tile 0/0/0 in world_cities.mbtiles
			format: PBF, data: "1f8b0800000000000203", tilesize: 512,
		},
	}

	for _, tc := range tests {
		data, err := hex.DecodeString(tc.data)
		if err != nil {
			t.Error("Error decoding hex image data", err)
		}

		tilesize, err := detectTileSize(tc.format, data)
		if err != nil {
			t.Error("Error detecting tile size: ", err)
		}
		if tilesize != tc.tilesize {
			t.Error("Tile size", tilesize, "does not match expected value", tc.tilesize)
		}
	}
}
