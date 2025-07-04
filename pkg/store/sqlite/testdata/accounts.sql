-- account records
INSERT INTO accounts (id,customer_id,first_name,last_name,travel_address,ivms101,created,modified) VALUES
    -- SQL NULL IVMS101 Record
    (x'018ECD7C995324EB921AE98B9676EAD8',39425390,'Frank','Westman','taaV7TSR6527onDWiSyoiNAzE2QE87JcwXvcJ9UZeWjUXPUCBScAMNkKCFbMyhCMVdXH9REo1gfA2c3DTm46En7JRYHz5QZZWDUbUDg',null,'2024-04-11T14:07:58+00:00','2024-04-11T14:07:58+00:00'),
    -- JSON null IVMS101 Record
    (x'018ECD8D811EAE0508A28E73D9B25675',27166869,'Mary','Tilcott','taaV7TSR6527onDWiSyoiNAzE2QE87JcwXvcJ9UZeWjUXPUCBVXFh7PaKtYcmSWEAUnxjMLENv43R51zLrF56JBXW2bJB9XoRfzWnip',x'6E756C6C','2024-04-11T14:26:26+00:00','2024-04-11T14:26:26+00:00'),
    -- JSON b64 IVMS101 Record (ID: "01JXWZTAJ64YTC34T5N47J7H2V")
    (x'019779fd2a4627b4c19345a90f23c45b',28282828,'Julius','Novachrono','taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYpfZUq4NiGEmrvTFrLAv194tgWHZyNjaWasCEX4P3NHcpHSXk8vyDTU5FBV','eyJuYXR1cmFsUGVyc29uIjp7Im5hbWUiOnsibmFtZUlkZW50aWZpZXIiOlt7InByaW1hcnlJZGVudGlmaWVyIjoiSnVsaXVzIiwic2Vjb25kYXJ5SWRlbnRpZmllciI6Ik5vdmFjaHJvbm8iLCJuYW1lSWRlbnRpZmllclR5cGUiOiJMRUdMIn1dfX19','2025-06-16T08:28:10-10:00','2025-06-16T08:28:10-10:00'),
    -- JSON txt IVMS101 Record (ID: "01JXX0D2X0XWFPTCJ0TCBQGKN3") (with milliseconds timestamps)
    (x'01977a068ba0ef1f6d3240d317784ea3',27683,'Yami','Sukehiro','taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYpfZfTNeBJRb79ZyoVof538YqcVSkhbPioDf7nXDjGVRuQfhWVp44r9Z3Db','{"naturalPerson":{"name":{"nameIdentifier":[{"primaryIdentifier":"Sukehiro","secondaryIdentifier":"Sukehiro","nameIdentifierType":"LEGL"}]}}}',"2025-06-16T08:31:48.427458-10:00","2025-06-16T08:31:48.427458-10:00"),
    -- Protocol buffer IVMS101 Record (ID: "01JXX0FBBJM8KTEPZZBRWJBC14") (with milliseconds timestamps)
    (x'01977a07ad72a227a75bff5e3925b024',999999,'Rakuro','Hizutome','taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYpfZfTgGWZEEA4SyrJyxSr6GwiAfu4hXS9Af45C5TXyFvPDaR5KTsPpWkz5','ChgKFgoUCgZSYWt1cm8SCEhpenV0b21lGAQ=','2025-06-16T08:41:04.470026-10:00','2025-06-16T08:41:04.470026-10:00')
;

-- crypto address records
INSERT INTO crypto_addresses (id,account_id,crypto_address,network,asset_type,tag,travel_address,created,modified) VALUES
    (x'018ECD7C995324EB921AE98BE2B75D56',x'018ECD7C995324EB921AE98B9676EAD8','n2irvV1QpYfV2XysspZ9hdiQyHHHh8xtX3','BTC',null,null,'ta8b1ZGjAJgDsUTrKuZMZ2dFKozNYyfAhhBMZRgnU9z5RKDcesjr2K3sCcJcEczDteRwZjy9rCToKPwxMvEzyQrbu3Av7VRbQKjAYg','2024-04-11T14:07:58+00:00','2024-04-11T14:25:17+00:00'),
    (x'018ECD8D811EAE0508A28E74D676C45C',x'018ECD8D811EAE0508A28E73D9B25675','0x64FFD67A858C013E0EBED36B9ECD0E77C376AF26','ETH',null,null,'ta8b1ZGjAJgDsUTrKuZMZ2dFKozNYyfAhhBMZRgnU9z5RKDchnqBkxJzqZZPyvrtjv7eYzKC2N1XPEnSyuCaQwWBimosBjRxtY1xsy','2024-04-11T14:26:26+00:00','2024-04-11T14:45:46+00:00'),
    (x'018ECD8D811F5A4058F06ED166DC4B96',x'018ECD8D811EAE0508A28E73D9B25675','mjJ9xufmdSfZLRUXV6Ac3r64M6bbrxCu48','LTC',null,null,'ta8b1ZGjAJgDsUTrKuZMZ2dFKozNYyfAhhBMZRgnU9z5RKDchnqBkxjLMipN6KW1oGYKqNQ7JnLYnB7DYcyzWwfFaafpHBCS4uaHZY','2024-04-11T14:26:26+00:00','2024-04-11T14:26:26+00:00')
;
