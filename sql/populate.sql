CREATE TABLE IF NOT EXISTS "public"."pictures" (
    "id" integer NOT NULL,
    "url" "text" NOT NULL,
    "title" "text",
    "author" "text",
    "permalink" "text",
    "score" integer,
    "nsfw" boolean,
    "greyscale" boolean,
    "time" integer,
    "width" integer,
    "height" integer,
    "sprocket" boolean DEFAULT false,
    "lowurl" "text",
    "lowwidth" integer,
    "lowheight" integer,
    "medurl" "text",
    "medwidth" integer,
    "medheight" integer,
    "highurl" "text",
    "highwidth" integer,
    "highheight" integer
);

CREATE SEQUENCE "public"."pictures_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE "public"."pictures_id_seq" OWNED BY "public"."pictures"."id";


ALTER TABLE ONLY "public"."pictures" ALTER COLUMN "id" SET DEFAULT "nextval"('"public"."pictures_id_seq"'::"regclass");


COPY "public"."pictures" ("id", "url", "title", "author", "permalink", "score", "nsfw", "greyscale", "time", "width", "height", "sprocket", "lowurl", "lowwidth", "lowheight", "medurl", "medwidth", "medheight", "highurl", "highwidth", "highheight") FROM stdin;
1764	https://d3i73ktnzbi69i.cloudfront.net/a057c1b8-b7a9-4ff3-95c4-6252d8215ffc.jpeg	A way out / Canon AE1P / 50mm 1.4 / Lomography 800	u/supersoldierpeek	https://www.reddit.com/r/analog/comments/t8382p/a_way_out_canon_ae1p_50mm_14_lomography_800/	326	f	f	1646586478	2112	2640	f	https://d3i73ktnzbi69i.cloudfront.net/e598323a-7f0f-4219-98d6-56b02d335722.jpeg	256	320	https://d3i73ktnzbi69i.cloudfront.net/a5001fa7-3fc4-4471-9191-0579a2190bf5.jpeg	614	768	https://d3i73ktnzbi69i.cloudfront.net/b6a90733-21ec-4b8f-bb7b-9adab71e2a31.jpeg	960	1200
1765	https://d3i73ktnzbi69i.cloudfront.net/55b6bb8b-77e6-48aa-99a3-f5e1c946353f.jpeg	Big Sur [Hasselblad 500cm, Zeiss 80 f/2.8, Portra 800]	u/jshank20	https://www.reddit.com/r/analog/comments/t85rsl/big_sur_hasselblad_500cm_zeiss_80_f28_portra_800/	179	f	f	1646593442	2505	2505	f	https://d3i73ktnzbi69i.cloudfront.net/755a4c87-b904-4a28-a67d-f31d09cf3604.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/b47b14a6-d28e-41b0-880c-29c3d726727f.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/b31884c9-20e6-4824-987d-b58dbd909258.jpeg	1200	1200
1766	https://d3i73ktnzbi69i.cloudfront.net/191dbb1d-dcd4-44c0-8323-53059a7b08e2.jpeg	Kentmere Pan 400.	u/_INLIVINGCOLOR	https://www.reddit.com/r/analog_bw/comments/t8ay8m/kentmere_pan_400/	5	f	t	1646607854	1024	1545	f	https://d3i73ktnzbi69i.cloudfront.net/83dace73-4b0b-44b2-8138-bb5daba0d167.jpeg	212	320	https://d3i73ktnzbi69i.cloudfront.net/9c2ccef7-eaad-441d-92c4-cb5186b83b98.jpeg	509	768	https://d3i73ktnzbi69i.cloudfront.net/2900129c-6e56-41ed-8524-e086d14fa0e2.jpeg	795	1200
1767	https://d3i73ktnzbi69i.cloudfront.net/61ea8cb8-983e-4268-9945-ccfd05116211.jpeg	4 [Mamiya 6 | 75 3.5 | Portra 160]	u/sillo38	https://www.reddit.com/r/analog/comments/t8pn11/4_mamiya_6_75_35_portra_160/	1559	f	f	1646661011	3200	3180	f	https://d3i73ktnzbi69i.cloudfront.net/1280a9d2-a5ae-4a82-b885-d396f10e6a45.jpeg	320	318	https://d3i73ktnzbi69i.cloudfront.net/98a7d886-b9cf-4319-b47c-f12a4c51f703.jpeg	768	763	https://d3i73ktnzbi69i.cloudfront.net/39a0def3-bdb9-4816-aff4-168c99716811.jpeg	1200	1193
1768	https://d3i73ktnzbi69i.cloudfront.net/79a65d90-0b3e-45fc-a4ea-cc614c3c8994.jpeg	bird creek [mamiya rb67, 180/4.5, kodak gold 100]	u/llllllllllllogan	https://www.reddit.com/r/analog/comments/t8rp5c/bird_creek_mamiya_rb67_18045_kodak_gold_100/	479	f	f	1646666878	4572	2036	f	https://d3i73ktnzbi69i.cloudfront.net/c189156f-78f4-4cc0-adc7-f44e9adf8243.jpeg	320	143	https://d3i73ktnzbi69i.cloudfront.net/935f8826-368a-40a2-b76a-3384bb88fb79.jpeg	768	342	https://d3i73ktnzbi69i.cloudfront.net/5b2f432d-b57a-4c89-ba34-d2d5ecb22c6b.jpeg	1200	534
1769	https://d3i73ktnzbi69i.cloudfront.net/3e193374-fe47-4d81-a801-01262cebc925.jpeg	Manzanita, OR - Ektar 100 // Pentax 6x7 // 105mm	u/the_juliette_show	https://www.reddit.com/r/analog/comments/t8jo2x/manzanita_or_ektar_100_pentax_6x7_105mm/	1661	f	f	1646637641	1440	1750	f	https://d3i73ktnzbi69i.cloudfront.net/60f6bc7e-8220-462f-bbb4-8f31de3162f5.jpeg	263	320	https://d3i73ktnzbi69i.cloudfront.net/85037466-6eeb-43bc-89cf-0a1ebbd589ec.jpeg	632	768	https://d3i73ktnzbi69i.cloudfront.net/1afd08ba-c028-4809-b327-605b7bdbf449.jpeg	987	1200
1770	https://d3i73ktnzbi69i.cloudfront.net/a8332042-4d45-408f-8aa2-1999fbb08915.jpeg	Aurora borealis Norway - Tromsø || Mamiya7 - 65mm f4 || Portra800	u/medialer_murks	https://www.reddit.com/r/analog/comments/t8wel4/aurora_borealis_norway_tromsø_mamiya7_65mm_f4/	180	f	f	1646679181	5000	4026	f	https://d3i73ktnzbi69i.cloudfront.net/0f563503-5c64-4d3b-a918-09612e72dbd9.jpeg	320	258	https://d3i73ktnzbi69i.cloudfront.net/817f06d0-69f3-4e61-865b-6cd7e752ecde.jpeg	768	618	https://d3i73ktnzbi69i.cloudfront.net/31fc7d8a-94ad-458d-abfe-6ebbd999d571.jpeg	1200	966
1771	https://d3i73ktnzbi69i.cloudfront.net/0e7295b7-100c-4cd0-acee-b184304ddcdd.jpeg	J / Pentax 67, 90mm, HP5, darkroom print	u/jacckowski	https://www.reddit.com/r/analog/comments/t8pqlw/j_pentax_67_90mm_hp5_darkroom_print/	397	t	t	1646661328	2327	3000	f	https://d3i73ktnzbi69i.cloudfront.net/aac5e5f7-cc4d-4174-ba51-b11f5e610b87.jpeg	248	320	https://d3i73ktnzbi69i.cloudfront.net/33c82e24-fc6b-48d8-b231-3884a81bec61.jpeg	596	768	https://d3i73ktnzbi69i.cloudfront.net/322f0207-36e6-4a3f-af43-48d55febb67e.jpeg	931	1200
1772	https://d3i73ktnzbi69i.cloudfront.net/1e8217d0-47de-4666-88a0-052755d24430.jpeg	New Brunswick / Fomapan 100	u/ry_ta506	https://www.reddit.com/r/analog_bw/comments/t8rilz/new_brunswick_fomapan_100/	33	f	t	1646666387	2554	1694	f	https://d3i73ktnzbi69i.cloudfront.net/632f80e2-681a-4095-b96d-4f7cf1255c48.jpeg	320	212	https://d3i73ktnzbi69i.cloudfront.net/aef41149-3ee1-4cc0-b906-83d8f6032ea4.jpeg	768	509	https://d3i73ktnzbi69i.cloudfront.net/7ad1d36c-8c18-4e95-a4a1-fb4619909e5a.jpeg	1200	796
1778	https://d3i73ktnzbi69i.cloudfront.net/193fe5f2-2c2e-437b-b21b-8636e300e3cb.jpeg	Nikon N80 + Cinestill 800, accidental triple exposure in Philly	u/StylesFieldstone	https://www.reddit.com/r/analog/comments/t9my0k/nikon_n80_cinestill_800_accidental_triple/	1366	f	f	1646762946	2433	3637	f	https://d3i73ktnzbi69i.cloudfront.net/ddb654e1-ee5c-4d10-af11-fb994e959530.jpeg	214	320	https://d3i73ktnzbi69i.cloudfront.net/d9072c51-7192-4cb2-adcf-b5cefabf582e.jpeg	514	768	https://d3i73ktnzbi69i.cloudfront.net/82673be9-4883-4ec5-a194-50e12699338b.jpeg	803	1200
1779	https://d3i73ktnzbi69i.cloudfront.net/ac26d4eb-e0be-4fb1-8971-cf330bf219f2.jpeg	"Enlightenment" (2021, Mamiya RB67 Pro-S + Mamiya-Sekor C 65mm, Kodak Portra 800@1600)	u/Koneser_fotografii	https://www.reddit.com/r/analog/comments/t9j2c1/enlightenment_2021_mamiya_rb67_pros_mamiyasekor_c/	242	t	f	1646752813	2048	2048	f	https://d3i73ktnzbi69i.cloudfront.net/273561a4-bcb4-40b1-9061-d09dbb7483dd.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/132534fe-fba0-49d1-b4c1-c733057e1f15.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/7f5568dd-95db-4f3a-9ea1-047984886f68.jpeg	1200	1200
1780	https://d3i73ktnzbi69i.cloudfront.net/88c68576-d4c5-4677-9eb1-eaa548ef35fd.jpeg	Cumberland beach (Mamiya 645. 80mm, Portra 400). Does this look level?	u/kurtozan251	https://www.reddit.com/r/analog/comments/t9kmmu/cumberland_beach_mamiya_645_80mm_portra_400_does/	161	f	f	1646756894	1467	2000	f	https://d3i73ktnzbi69i.cloudfront.net/8e85ccb3-0672-4336-b6a2-1680e4b9b5d2.jpeg	235	320	https://d3i73ktnzbi69i.cloudfront.net/4878ff14-4b7a-49ce-a844-35708e60b9ce.jpeg	563	768	https://d3i73ktnzbi69i.cloudfront.net/0530f43b-6b7b-475a-abed-bfda526c4cf4.jpeg	880	1200
1781	https://d3i73ktnzbi69i.cloudfront.net/5c91d1e1-0e15-4459-a081-1b54cdef233e.jpeg	The Betrayal of Gravity [Ilford FP4]	u/maxcooperavl	https://www.reddit.com/r/analog_bw/comments/t9l2tz/the_betrayal_of_gravity_ilford_fp4/	21	f	t	1646758090	1440	1800	f	https://d3i73ktnzbi69i.cloudfront.net/d480bf45-9464-43f3-a7ce-a9e79dae2b96.jpeg	256	320	https://d3i73ktnzbi69i.cloudfront.net/829cd485-dc1c-4fbd-91c9-af61f2161208.jpeg	614	768	https://d3i73ktnzbi69i.cloudfront.net/c315fe7c-6dda-46b6-b14c-2071ea5f18ce.jpeg	960	1200
1782	https://d3i73ktnzbi69i.cloudfront.net/9bf0180a-8f06-4acb-993d-ff34a2b9baa4.jpeg	France [Zenza Bronica SQ-A | Zenzanon 80mm f/2.8 | Kodak ColorPlus 200]	u/MichaWha	https://www.reddit.com/r/SprocketShots/comments/t9rquw/france_zenza_bronica_sqa_zenzanon_80mm_f28_kodak/	8	f	f	1646775980	1200	1924	t	https://d3i73ktnzbi69i.cloudfront.net/db2304a8-36cc-40d8-a76e-af47adf5853c.jpeg	200	320	https://d3i73ktnzbi69i.cloudfront.net/71e5ab6f-df01-441e-bbaa-453b3868dc02.jpeg	479	768	https://d3i73ktnzbi69i.cloudfront.net/1c7d6cb1-cd8f-4d41-9da9-4db46c4887bb.jpeg	748	1200
1773	https://d3i73ktnzbi69i.cloudfront.net/a5329f62-f992-4907-b55e-d85361789da8.jpeg	Hp5+	u/Falevian	https://www.reddit.com/r/analog_bw/comments/t91bw2/hp5/	10	f	t	1646691981	4096	4096	f	https://d3i73ktnzbi69i.cloudfront.net/8813e1b5-163c-456c-9968-214cc40ac4db.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/51a1fdc2-e2db-4b9f-a29b-c6b8011299b8.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/55e5f2c2-6675-4181-9812-b9cb96f3d879.jpeg	1200	1200
1783	https://d3i73ktnzbi69i.cloudfront.net/5cf736e4-3d14-4b11-aef0-0b04794ba8b6.jpeg	farmland - Pentax 67 - 90mm 2.8	u/dthomp27	https://www.reddit.com/r/analog/comments/t9wro7/farmland_pentax_67_90mm_28/	146	f	f	1646790897	1862	2517	f	https://d3i73ktnzbi69i.cloudfront.net/ac0f0b54-7c5f-4195-8679-07ea91492f23.jpeg	237	320	https://d3i73ktnzbi69i.cloudfront.net/24f7db28-10b0-4dff-ad1c-f0375e1863f9.jpeg	568	768	https://d3i73ktnzbi69i.cloudfront.net/240a12bc-bf50-4594-a6db-30d70c00ead9.jpeg	888	1200
1784	https://d3i73ktnzbi69i.cloudfront.net/df00453c-c54d-4c86-8389-875a52d7b278.jpeg	Rodanthe Pier ■ Pentax 67 ■ 55mm f/4 ■ Portra 400	u/peterncsu	https://www.reddit.com/r/analog/comments/t9yy4n/rodanthe_pier_pentax_67_55mm_f4_portra_400/	67	f	f	1646797974	4502	3600	f	https://d3i73ktnzbi69i.cloudfront.net/e57655f1-660f-4c66-a950-d79da743741b.jpeg	320	256	https://d3i73ktnzbi69i.cloudfront.net/c8a219aa-5b67-465e-ab99-63d32a627a51.jpeg	768	614	https://d3i73ktnzbi69i.cloudfront.net/dc0bb0c5-0ffd-4c0f-80a0-596328a0cd60.jpeg	1200	960
1785	https://d3i73ktnzbi69i.cloudfront.net/d40f7109-3dfe-4724-9e2e-dde1cf2c8235.jpeg	Torn Net • Mamiya RZ67 • 180mm • Portra 160	u/bocceboy95	https://www.reddit.com/r/analog/comments/t9vk0j/torn_net_mamiya_rz67_180mm_portra_160/	98	f	f	1646787044	2589	3322	f	https://d3i73ktnzbi69i.cloudfront.net/e2f572fd-01e6-4356-8ede-7046f7f2f193.jpeg	249	320	https://d3i73ktnzbi69i.cloudfront.net/f9970aa1-562e-484f-9438-9dd43fb90113.jpeg	599	768	https://d3i73ktnzbi69i.cloudfront.net/66ca63af-c357-45b7-ab2f-3183a99a22cd.jpeg	935	1200
2058	https://d3i73ktnzbi69i.cloudfront.net/bbcdc356-d136-479a-aab8-3187a110f178.jpeg	I just like spiral stairs [Zenza Bronica SQ-A | Zenzanon 80mm f/2.8 | Kodak Ektachrome 100]	u/MichaWha	https://www.reddit.com/r/SprocketShots/comments/tu0o2d/i_just_like_spiral_stairs_zenza_bronica_sqa/	13	f	f	1648844963	1100	1767	t	https://d3i73ktnzbi69i.cloudfront.net/365b5fd6-ed31-4c4a-8d95-f06a8d3f769b.jpeg	199	320	https://d3i73ktnzbi69i.cloudfront.net/1fd629e2-9697-41a9-9794-e48ede2b21e1.jpeg	478	768	https://d3i73ktnzbi69i.cloudfront.net/466f82e2-071b-41ef-bcc1-b2a6c91db79f.jpeg	747	1200
2107	https://d3i73ktnzbi69i.cloudfront.net/633f96f6-d431-492d-8a87-20aa41409ae6.jpeg	More Pentax Auto 110, Pentax 24mm f2.8, photos on expired Lomography Orca film, little package, big punch!	u/jochno	https://www.reddit.com/r/analog/comments/txxaik/more_pentax_auto_110_pentax_24mm_f28_photos_on/	144	f	t	1649281928	3035	2365	f	https://d3i73ktnzbi69i.cloudfront.net/a86c9a1d-ad56-4211-ab6e-1c1e9c8c55bc.jpeg	320	249	https://d3i73ktnzbi69i.cloudfront.net/9099af71-78de-4177-bb1d-ff2befe723a4.jpeg	768	598	https://d3i73ktnzbi69i.cloudfront.net/cc12784d-c1c9-43c1-9560-1ab87c937c3c.jpeg	1200	935
2128	https://d3i73ktnzbi69i.cloudfront.net/489f61ac-f775-4e08-8354-997679a24e3c.jpeg	Harbour views // Canon F-1 // Fujifilm Provia 100F	u/photocactus	https://www.reddit.com/r/analog/comments/tzj0jj/harbour_views_canon_f1_fujifilm_provia_100f/	471	f	f	1649469780	3659	3956	f	https://d3i73ktnzbi69i.cloudfront.net/07c8e44d-fff1-409f-b332-492c41bff1e5.jpeg	296	320	https://d3i73ktnzbi69i.cloudfront.net/dcc08866-d6e3-4567-99d4-4eeca8270d56.jpeg	710	768	https://d3i73ktnzbi69i.cloudfront.net/192c50af-39b2-4c8c-ae40-3ec837ff3ef6.jpeg	1110	1200
2129	https://d3i73ktnzbi69i.cloudfront.net/26b654b6-3abc-4a61-a8c8-045fdc05fc3b.jpeg	Cannon Beach|Nikon F4 20mm 2.8|Kodak Gold 200	u/Amocat_Tacoma	https://www.reddit.com/r/analog/comments/tzhgyl/cannon_beachnikon_f4_20mm_28kodak_gold_200/	158	f	f	1649464548	1080	1186	f	https://d3i73ktnzbi69i.cloudfront.net/44212e67-c57b-4db5-94cd-53d249dee893.jpeg	291	320	https://d3i73ktnzbi69i.cloudfront.net/a21c34a1-efd6-4b56-a780-8d15cf489b50.jpeg	699	768	https://d3i73ktnzbi69i.cloudfront.net/466186c3-040d-434f-a57a-cd088bb5b44d.jpeg	1080	1186
2130	https://d3i73ktnzbi69i.cloudfront.net/be409ed1-699d-4d5d-b6da-b6578a26ec80.jpeg	Pines [Pentax Spotmatic | 50mm 1.8 | Kodak Vision 200T]	u/nguyentritai2906	https://www.reddit.com/r/analog/comments/tzl507/pines_pentax_spotmatic_50mm_18_kodak_vision_200t/	68	f	f	1649477155	2397	3543	f	https://d3i73ktnzbi69i.cloudfront.net/ec595f54-812b-4a82-bb32-cd3cfb43d0b3.jpeg	216	320	https://d3i73ktnzbi69i.cloudfront.net/af4a9f67-88a6-436f-a8f7-3c8ef30cc0f9.jpeg	520	768	https://d3i73ktnzbi69i.cloudfront.net/a8dcca97-c72d-4434-84ab-07aff031dbe0.jpeg	812	1200
2131	https://d3i73ktnzbi69i.cloudfront.net/24b3d876-12c1-4136-843d-9f4f7e97e473.jpeg	Window gazing [Canon Elan 7E | Canon EF 50mm | Portra 400]	u/ohseephotography	https://www.reddit.com/r/analog/comments/tzhklj/window_gazing_canon_elan_7e_canon_ef_50mm_portra/	131	t	f	1649464908	5073	3401	f	https://d3i73ktnzbi69i.cloudfront.net/2fb44dc5-b9ed-4cb5-8550-0e3ca4ae928e.jpeg	320	215	https://d3i73ktnzbi69i.cloudfront.net/acb03953-e7cc-4ede-9c6d-d20439e25bcf.jpeg	768	515	https://d3i73ktnzbi69i.cloudfront.net/04605059-6fb7-4568-aa64-bef86c38600f.jpeg	1200	804
2140	https://d3i73ktnzbi69i.cloudfront.net/8faa5f15-5e26-41cb-9f05-03348bf48a4a.jpeg	Yashica MF-3 | Superia 400	u/salabin	https://www.reddit.com/r/analog/comments/u071c4/yashica_mf3_superia_400/	138	f	f	1649553814	1080	1351	f	https://d3i73ktnzbi69i.cloudfront.net/5925acb4-e270-4fd6-81f5-1c47ca62a50e.jpeg	256	320	https://d3i73ktnzbi69i.cloudfront.net/391f4c41-7063-49d7-91f3-2f0319e1ab0c.jpeg	614	768	https://d3i73ktnzbi69i.cloudfront.net/e6d0a4b0-995b-4ed2-bd2e-a26859137ee0.jpeg	959	1200
2141	https://d3i73ktnzbi69i.cloudfront.net/4a1ec14c-cb23-424f-b861-225e55fcfaec.jpeg	Cook-Out [Hasselblad 201f, Zeiss 110 f2, Portra 400]	u/jshank20	https://www.reddit.com/r/analog/comments/u07iwo/cookout_hasselblad_201f_zeiss_110_f2_portra_400/	76	f	f	1649555546	2508	2508	f	https://d3i73ktnzbi69i.cloudfront.net/d89bf5dc-cc9e-49a2-9c18-e627ebc6c133.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/21a3a831-3b77-444c-bf47-6ea225f08549.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/74a446fe-05da-409a-922c-8ee80473904b.jpeg	1200	1200
2147	https://d3i73ktnzbi69i.cloudfront.net/41141365-651b-40a9-bfd3-0e44b556436a.jpeg	Orange Ferrari | Hasselblad 500c/m | 80mm F2.8 | Velvia 50	u/vese	https://www.reddit.com/r/analog/comments/u0osmb/orange_ferrari_hasselblad_500cm_80mm_f28_velvia_50/	499	f	f	1649618359	4755	4755	f	https://d3i73ktnzbi69i.cloudfront.net/b4757ddb-a85a-47b8-a46e-20772fe41027.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/26fc8f5e-f133-4d20-9a94-6ed6981a5bd9.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/f4292a3b-f7d0-4687-a1a7-44fa04635a5f.jpeg	1200	1200
2148	https://d3i73ktnzbi69i.cloudfront.net/e7406a05-149e-4f39-90dd-e88bdfb98771.jpeg	Isa | Polaroid one step + | iType Color	u/TheBrushCreative	https://www.reddit.com/r/analog/comments/u0l1r4/isa_polaroid_one_step_itype_color/	374	f	f	1649607822	2079	2528	f	https://d3i73ktnzbi69i.cloudfront.net/7198cd5b-2722-4f8a-8c43-fa5a33480167.jpeg	263	320	https://d3i73ktnzbi69i.cloudfront.net/97190298-1c98-41ea-9e28-afce7bb29970.jpeg	632	768	https://d3i73ktnzbi69i.cloudfront.net/6f6265bf-b72e-46e8-a782-1a74bb6bb268.jpeg	987	1200
1774	https://d3i73ktnzbi69i.cloudfront.net/69208667-f150-4c88-bca0-aae5fb8f743c.jpeg	Double Exposure on 35mm Ektar	u/StylesFieldstone	https://www.reddit.com/r/analog/comments/t969px/double_exposure_on_35mm_ektar/	1414	f	f	1646706549	2400	3588	f	https://d3i73ktnzbi69i.cloudfront.net/41bc0d77-fadf-40cf-9893-1b64febc3863.jpeg	214	320	https://d3i73ktnzbi69i.cloudfront.net/7af06926-4844-47b5-a371-d260af1b447d.jpeg	514	768	https://d3i73ktnzbi69i.cloudfront.net/e6398786-da1c-4fb5-8fcb-d64e89ab0cae.jpeg	803	1200
1775	https://d3i73ktnzbi69i.cloudfront.net/3f9b4f38-91c9-40a5-aa18-12624ab977ed.jpeg	Death Valley 35mm	u/StylesFieldstone	https://www.reddit.com/r/analog/comments/t96dl6/death_valley_35mm/	410	f	f	1646706872	2433	3637	f	https://d3i73ktnzbi69i.cloudfront.net/37bd53e0-0a54-4dad-9f5a-6c1b8cf79c93.jpeg	214	320	https://d3i73ktnzbi69i.cloudfront.net/d683637c-f7e5-4692-8fa3-2917183a1d3c.jpeg	514	768	https://d3i73ktnzbi69i.cloudfront.net/c2017119-e8e9-46c8-883c-6e5374e157d3.jpeg	803	1200
1786	https://d3i73ktnzbi69i.cloudfront.net/fba1e091-64ef-4498-88da-e34edc04dc41.jpeg	Lofoten, Norway [Olympus Trip 35mm Fujicolor C200]	u/TQairstrike	https://www.reddit.com/r/analog/comments/ta3etl/lofoten_norway_olympus_trip_35mm_fujicolor_c200/	1087	f	f	1646814939	2433	3637	f	https://d3i73ktnzbi69i.cloudfront.net/c38bafcc-9840-4e94-b73d-beca8fd4fa6a.jpeg	214	320	https://d3i73ktnzbi69i.cloudfront.net/70c74f90-2468-457a-ad23-d4ae4e23a826.jpeg	514	768	https://d3i73ktnzbi69i.cloudfront.net/8ecff248-7522-4a88-ab83-6eb9d636febe.jpeg	803	1200
1787	https://d3i73ktnzbi69i.cloudfront.net/6b43074a-7191-43f6-bb98-44d565552ce3.jpeg	"Old bird" fujica st605, fujion 55mm, kodak gold 200	u/marshmilf	https://www.reddit.com/r/analog/comments/ta6hsv/old_bird_fujica_st605_fujion_55mm_kodak_gold_200/	138	f	f	1646828008	1838	1238	f	https://d3i73ktnzbi69i.cloudfront.net/27baaef7-6c13-491d-b08f-8f057473e0a2.jpeg	320	216	https://d3i73ktnzbi69i.cloudfront.net/f9848442-dbbe-4482-8475-bdace157eaa9.jpeg	768	517	https://d3i73ktnzbi69i.cloudfront.net/c3af3a35-da7d-43e3-831a-ba3adf34205e.jpeg	1200	808
1788	https://d3i73ktnzbi69i.cloudfront.net/91d51ed4-16b6-4fb5-b493-bbfc58e80299.jpeg	Dry Dock, Sunderland | Hasselblad 500 C/M | 50mm Distagon | RTP II	u/martintype	https://www.reddit.com/r/analog/comments/ta3yws/dry_dock_sunderland_hasselblad_500_cm_50mm/	135	f	f	1646817417	793	800	f	https://d3i73ktnzbi69i.cloudfront.net/830687ad-703a-410f-a50e-7d5d09b68f91.jpeg	317	320	https://d3i73ktnzbi69i.cloudfront.net/f7b4a8fb-cb02-4f20-a6ba-47bfacb2ff61.jpeg	761	768	https://d3i73ktnzbi69i.cloudfront.net/b2a4ee6c-0da8-49fe-980b-72422413bd1c.jpeg	793	800
1789	https://d3i73ktnzbi69i.cloudfront.net/770015b3-e476-46ed-a733-d043c4c12d9e.jpeg	Canon Autoboy + Ektar 100	u/hidemelon	https://www.reddit.com/r/analog/comments/ta7jwk/canon_autoboy_ektar_100/	58	f	f	1646831761	1818	1228	f	https://d3i73ktnzbi69i.cloudfront.net/2e6b3cf0-8a17-44fb-80b7-d928b2c5a048.jpeg	320	216	https://d3i73ktnzbi69i.cloudfront.net/b6d2f5ec-f82d-4759-9d60-536bf1063cce.jpeg	768	519	https://d3i73ktnzbi69i.cloudfront.net/73b6fbba-a546-42ed-8188-409bfa7f3c84.jpeg	1200	811
1790	https://d3i73ktnzbi69i.cloudfront.net/da801ca4-9635-4255-91c2-735cbffbdbcb.jpeg	Somewhere below freezing | Lomography 800 | Canon 1V | 50mm 1.4	u/navazuals	https://www.reddit.com/r/analog/comments/tadcqu/somewhere_below_freezing_lomography_800_canon_1v/	1399	f	f	1646848190	2955	2000	f	https://d3i73ktnzbi69i.cloudfront.net/adf3f2c2-87e3-4c5c-b188-5cf526bd855a.jpeg	320	217	https://d3i73ktnzbi69i.cloudfront.net/0c182f43-78a1-474b-b6e2-deb79b828348.jpeg	768	520	https://d3i73ktnzbi69i.cloudfront.net/4bb3deef-cadd-4aa7-a0be-abd49cc60eb1.jpeg	1200	812
1791	https://d3i73ktnzbi69i.cloudfront.net/11d95c99-0235-427e-b762-eed0bbbe57bb.jpeg	Rain Shadow | Hasselblad 500cm | Sonnar 150 | Portra 160	u/Just_InGrain	https://www.reddit.com/r/analog/comments/taee6a/rain_shadow_hasselblad_500cm_sonnar_150_portra_160/	730	f	f	1646851129	4890	4890	f	https://d3i73ktnzbi69i.cloudfront.net/815cad93-f207-4ed9-8d54-08f74afa89bd.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/cc9cc5b1-a08c-4ce4-9daf-541f88b1de67.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/8e731b32-ba13-4591-92bc-23441a545734.jpeg	1200	1200
1792	https://d3i73ktnzbi69i.cloudfront.net/ece14ea2-f31a-4af6-b075-6438608e1ba9.jpeg	First of the roll. Portra 800, mamiya645, 80mm Sekor C Lens, f2.8	u/whoisnkdeye	https://www.reddit.com/r/analog/comments/tafocf/first_of_the_roll_portra_800_mamiya645_80mm_sekor/	152	f	f	1646854637	3616	4711	f	https://d3i73ktnzbi69i.cloudfront.net/ef3024df-b77e-4b88-a4a5-f680aba30156.jpeg	246	320	https://d3i73ktnzbi69i.cloudfront.net/63e9082c-47bf-4b38-9a64-68855b6e6bcf.jpeg	589	768	https://d3i73ktnzbi69i.cloudfront.net/684a33e2-7bf2-41f5-9c38-159078c7365b.jpeg	921	1200
1793	https://d3i73ktnzbi69i.cloudfront.net/25c82941-d1cb-4434-bb52-10a11c02218b.jpeg	A Farmer, a crop, and a best friend. [Kodachrome, Unknown, 1959. - Epson V850 Pro]	u/SalmonSnail	https://www.reddit.com/r/analog/comments/tadtgf/a_farmer_a_crop_and_a_best_friend_kodachrome/	119	f	f	1646849432	7211	4881	f	https://d3i73ktnzbi69i.cloudfront.net/7e6958f4-f8a2-432b-9bee-eb8227e98cb4.jpeg	320	217	https://d3i73ktnzbi69i.cloudfront.net/89f1ef84-1cee-4001-abaa-194ee750fe9d.jpeg	768	520	https://d3i73ktnzbi69i.cloudfront.net/d34cfa6d-5f5e-4db8-84db-3635d7c7da78.jpeg	1200	812
1794	https://d3i73ktnzbi69i.cloudfront.net/a588aed9-2daa-4c01-989f-acd563ba831a.jpeg	I saw the truth in the curls of the vanishing girl [Kodak Tri-X 400]	u/sickestinvertebrate	https://www.reddit.com/r/analog_bw/comments/tadphc/i_saw_the_truth_in_the_curls_of_the_vanishing/	14	f	t	1646849121	1667	2500	f	https://d3i73ktnzbi69i.cloudfront.net/c3d5656d-bf24-4920-bb74-ef4d5cceb964.jpeg	213	320	https://d3i73ktnzbi69i.cloudfront.net/76105837-5aa2-4841-820b-fed9adbcdce7.jpeg	512	768	https://d3i73ktnzbi69i.cloudfront.net/c92fa593-ac38-48f0-8837-d87826c94f6f.jpeg	800	1200
1795	https://d3i73ktnzbi69i.cloudfront.net/996faa51-2705-4e5c-9616-025366d8fb18.jpeg	Te Henga to Whatipū [Nikon N80 / 50mm / Ektar 100]	u/AA_BATTERY	https://www.reddit.com/r/analog/comments/tap1jz/te_henga_to_whatipū_nikon_n80_50mm_ektar_100/	134	f	f	1646881785	4912	3257	f	https://d3i73ktnzbi69i.cloudfront.net/4ea8344c-dc65-4783-ad01-68fa27ca3a11.jpeg	320	212	https://d3i73ktnzbi69i.cloudfront.net/76e7b2ed-392e-4ffe-98db-9f2e08ed56a2.jpeg	768	509	https://d3i73ktnzbi69i.cloudfront.net/f04cf2f6-41e4-438a-a145-c168e7bb3e5a.jpeg	1200	796
1796	https://d3i73ktnzbi69i.cloudfront.net/084aedb1-57a7-461e-a3e7-3752739c3eca.jpeg	THIS IS WHY I BRING MY CAMERA EVERYWHERE | canon A1 | Ultramax 400 | 50mm 1.8	u/forfuckssakesbruv	https://www.reddit.com/r/analog/comments/tapr53/this_is_why_i_bring_my_camera_everywhere_canon_a1/	86	f	f	1646884084	1960	2941	f	https://d3i73ktnzbi69i.cloudfront.net/2f7c6571-0708-4464-b7ef-002bbc7467fd.jpeg	213	320	https://d3i73ktnzbi69i.cloudfront.net/6b81d9b4-52b4-4b19-815c-f0c2da81935d.jpeg	512	768	https://d3i73ktnzbi69i.cloudfront.net/16f53f40-7dae-49cc-b4e0-2ac532ee3883.jpeg	800	1200
1797	https://d3i73ktnzbi69i.cloudfront.net/e9db9f53-be0d-46d1-bb05-b7f724ed3e51.jpeg	Girl Through Glass. Rolleiflex 3.5. Portra 400	u/frauaike	https://www.reddit.com/r/analog/comments/taudpf/girl_through_glass_rolleiflex_35_portra_400/	710	f	f	1646901686	1658	1651	f	https://d3i73ktnzbi69i.cloudfront.net/3e8e1dce-8806-49f2-8084-d82ccf882fdd.jpeg	320	319	https://d3i73ktnzbi69i.cloudfront.net/a7800139-ebed-490f-bade-2cee7ce79c22.jpeg	768	765	https://d3i73ktnzbi69i.cloudfront.net/605e7452-686a-418b-a46d-d4afe9da1502.jpeg	1200	1195
1776	https://d3i73ktnzbi69i.cloudfront.net/16a1748d-03cc-47ba-82ea-d0fd4aae212c.jpeg	Shitbird [Canon A-1 + Portra 400]	u/azinza	https://www.reddit.com/r/analog/comments/t9fuqn/shitbird_canon_a1_portra_400/	182	f	f	1646742865	2320	3500	f	https://d3i73ktnzbi69i.cloudfront.net/9e18b20b-b8b9-4f69-bba1-295fc401e4f0.jpeg	212	320	https://d3i73ktnzbi69i.cloudfront.net/1218f320-f627-4a34-b850-ab8351cd45e7.jpeg	509	768	https://d3i73ktnzbi69i.cloudfront.net/adc66249-da4a-42d5-b39e-c86cdff8e034.jpeg	795	1200
1777	https://d3i73ktnzbi69i.cloudfront.net/a8fca753-0cff-496d-9f9a-c790879edfba.jpeg	Cool Cat. Gold 200, Minolta srT 200 , Rokkor-x 45mm	u/lfhooper	https://www.reddit.com/r/analog/comments/t9fq3f/cool_cat_gold_200_minolta_srt_200_rokkorx_45mm/	69	f	f	1646742391	1818	1228	f	https://d3i73ktnzbi69i.cloudfront.net/ace1806c-ab55-45ef-a8fc-c23e1ee5bddd.jpeg	320	216	https://d3i73ktnzbi69i.cloudfront.net/4ef4d077-6235-4025-9022-fe23982cf024.jpeg	768	519	https://d3i73ktnzbi69i.cloudfront.net/a73c47cf-5e81-41c2-a9f6-a7fbcf5771e6.jpeg	1200	811
1798	https://d3i73ktnzbi69i.cloudfront.net/35cb7851-8498-431e-9d11-17bb418a179f.jpeg	Leica M3 + Fujifilm iso 400	u/paranoid_drone	https://www.reddit.com/r/analog/comments/tayq0z/leica_m3_fujifilm_iso_400/	92	f	f	1646919107	4788	3175	f	https://d3i73ktnzbi69i.cloudfront.net/0d1d58c4-4573-4be9-b090-f2e653d96b35.jpeg	320	212	https://d3i73ktnzbi69i.cloudfront.net/71331da9-3e93-4ad9-bb2a-38af01e4ee4f.jpeg	768	509	https://d3i73ktnzbi69i.cloudfront.net/284c6781-c299-49ea-89e0-6136f4112d41.jpeg	1200	796
1799	https://d3i73ktnzbi69i.cloudfront.net/c3556a24-c502-41b0-92c9-f594480fa973.jpeg	Graduation Day | Canon Ftb | Portra 400	u/Ka_iru	https://www.reddit.com/r/analog/comments/taycpz/graduation_day_canon_ftb_portra_400/	83	f	f	1646917882	2331	1595	f	https://d3i73ktnzbi69i.cloudfront.net/09a9189d-541f-4118-ac4c-e6c0b123fb0a.jpeg	320	219	https://d3i73ktnzbi69i.cloudfront.net/236eff36-eba4-4570-b349-366e01750a8e.jpeg	768	526	https://d3i73ktnzbi69i.cloudfront.net/6e5884d6-d922-47bb-a738-45fffecb5e15.jpeg	1200	821
1800	https://d3i73ktnzbi69i.cloudfront.net/c0a096a2-6767-4e3f-b48d-22501df9c036.jpeg	At a café (Canon ae-1, takumar 28 f3.5, Colorplus)	u/enolaeid	https://www.reddit.com/r/analog/comments/tb09wz/at_a_café_canon_ae1_takumar_28_f35_colorplus/	57	f	f	1646923589	1890	1889	f	https://d3i73ktnzbi69i.cloudfront.net/cffe4637-ee07-40eb-aa03-644b15259a10.jpeg	320	320	https://d3i73ktnzbi69i.cloudfront.net/984240e8-c855-4a71-b23c-ce1a9dc33f2d.jpeg	768	768	https://d3i73ktnzbi69i.cloudfront.net/e84a651e-a26e-4921-b6ca-6c1e1922ecce.jpeg	1200	1199
2059	https://d3i73ktnzbi69i.cloudfront.net/a2ee5c0e-8979-41ff-9da5-d23847f79713.jpeg	100 years of storms - Pentax 67 55mm [Portra 400]	u/dthomp27	https://www.reddit.com/r/analog/comments/tu6kid/100_years_of_storms_pentax_67_55mm_portra_400/	635	f	f	1648861346	3465	2758	f	https://d3i73ktnzbi69i.cloudfront.net/440c201a-f9a9-48ec-892a-2bb8d0ef5247.jpeg	320	255	https://d3i73ktnzbi69i.cloudfront.net/65aae6a4-9258-41ac-a660-1debb6e11341.jpeg	768	611	https://d3i73ktnzbi69i.cloudfront.net/f6020930-7a45-4cb4-b360-2c97abe4d32b.jpeg	1200	955
2060	https://d3i73ktnzbi69i.cloudfront.net/2e576987-0165-489e-bc66-7c2373ad24aa.jpeg	This specific zone feels like an underwater amphitheater. With cliffs that drop off into 20m of water, all the sounds reverberate around us while we dive. It’s the end of whale season and we can still hear them singing when we hop in here. Nikonos v/hp5	u/cassec0u	https://www.reddit.com/r/analog/comments/tu4ocj/this_specific_zone_feels_like_an_underwater/	332	f	f	1648855625	2128	3296	f	https://d3i73ktnzbi69i.cloudfront.net/2fecb14d-6487-41f9-a99c-7c7113ea1620.jpeg	207	320	https://d3i73ktnzbi69i.cloudfront.net/f086397c-8c70-4ea5-a107-6d3bc829f876.jpeg	496	768	https://d3i73ktnzbi69i.cloudfront.net/a7e06e21-bb49-4ba7-8cf0-58b59ea4be12.jpeg	775	1200
2064	https://d3i73ktnzbi69i.cloudfront.net/0addd550-70a1-4536-8eea-bd3433fa07bb.jpeg	Girls in the kayak [canon Eos 33 Portra 160.Sigma art 18-35]	u/Alex_iwanski	https://www.reddit.com/r/analog/comments/tuqsuj/girls_in_the_kayak_canon_eos_33_portra_160sigma/	1643	t	f	1648927086	2400	3600	f	https://d3i73ktnzbi69i.cloudfront.net/ffe971fc-00a5-44e0-a7a4-b595de1187f6.jpeg	213	320	https://d3i73ktnzbi69i.cloudfront.net/24539759-ea54-48a6-b731-97418125baee.jpeg	512	768	https://d3i73ktnzbi69i.cloudfront.net/383a4bb8-5e0e-4b49-b0fb-e5b86f282238.jpeg	800	1200
2065	https://d3i73ktnzbi69i.cloudfront.net/83ca7c34-8893-41b8-89cb-b337f09aec68.jpeg	Blade Runner vibes [Canon AE-1 50mm f/1.4, CineStill 800T]	u/MxARC	https://www.reddit.com/r/analog/comments/tuohbd/blade_runner_vibes_canon_ae1_50mm_f14_cinestill/	1161	f	f	1648921031	2160	2331	f	https://d3i73ktnzbi69i.cloudfront.net/9d48f0a6-7578-4de2-9743-88b9434de9b0.jpeg	297	320	https://d3i73ktnzbi69i.cloudfront.net/86b8feb3-89f7-4bdc-bf1f-47992ef05827.jpeg	712	768	https://d3i73ktnzbi69i.cloudfront.net/586a1d74-85ac-4502-93fe-30ea43c38b01.jpeg	1112	1200
2066	https://d3i73ktnzbi69i.cloudfront.net/48fe60e6-a101-453b-804a-2ac80d257dc9.jpeg	Up on Melancholy Hill [Canon TLB / 28-55mm 3.5 / Portra 400]	u/sunnyintheoffice	https://www.reddit.com/r/analog/comments/tulrsv/up_on_melancholy_hill_canon_tlb_2855mm_35_portra/	1348	f	f	1648914195	1535	2290	f	https://d3i73ktnzbi69i.cloudfront.net/2fa8d749-38c4-4c08-b9d1-d8bd2019e16b.jpeg	214	320	https://d3i73ktnzbi69i.cloudfront.net/6e4e2d07-0c3d-41ed-aab1-3c4dad683a2f.jpeg	515	768	https://d3i73ktnzbi69i.cloudfront.net/b2b27f3a-4f08-4928-b15c-d3f2adef04f5.jpeg	804	1200
\.


SELECT pg_catalog.setval('"public"."pictures_id_seq"', 2572, true);


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "pictures_pkey" PRIMARY KEY ("id");


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "pictures_url_key" UNIQUE ("url");


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "unique_url" UNIQUE ("permalink");