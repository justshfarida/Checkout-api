--
-- PostgreSQL database dump
--

\restrict bSDNAUiFzfPMN6W1Lv0chFtfbjCssWTW4dGIUBYLUSVQetfN2d1tAojGDUA5gf5

-- Dumped from database version 18.4 (Debian 18.4-1.pgdg13+1)
-- Dumped by pg_dump version 18.4 (Debian 18.4-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: carts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.carts (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    item_id bigint NOT NULL,
    quantity bigint DEFAULT 1 NOT NULL,
    CONSTRAINT carts_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.carts OWNER TO postgres;

--
-- Name: carts_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.carts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.carts_id_seq OWNER TO postgres;

--
-- Name: carts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.carts_id_seq OWNED BY public.carts.id;


--
-- Name: items; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.items (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    price bigint NOT NULL,
    stock bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.items OWNER TO postgres;

--
-- Name: items_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.items_id_seq OWNER TO postgres;

--
-- Name: items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.items_id_seq OWNED BY public.items.id;


--
-- Name: line_items; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.line_items (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    item_id bigint NOT NULL,
    price bigint NOT NULL,
    quantity bigint NOT NULL,
    CONSTRAINT line_items_price_check CHECK ((price >= 0)),
    CONSTRAINT line_items_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.line_items OWNER TO postgres;

--
-- Name: line_items_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.line_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.line_items_id_seq OWNER TO postgres;

--
-- Name: line_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.line_items_id_seq OWNED BY public.line_items.id;


--
-- Name: orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.orders (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    total bigint NOT NULL,
    status character varying(20) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.orders OWNER TO postgres;

--
-- Name: orders_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.orders_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.orders_id_seq OWNER TO postgres;

--
-- Name: orders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.orders_id_seq OWNED BY public.orders.id;


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.refresh_tokens (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    token_hash bytea NOT NULL,
    expires_at timestamp without time zone NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.refresh_tokens OWNER TO postgres;

--
-- Name: refresh_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.refresh_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.refresh_tokens_id_seq OWNER TO postgres;

--
-- Name: refresh_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.refresh_tokens_id_seq OWNED BY public.refresh_tokens.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    email character varying(255) NOT NULL,
    hash bytea NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: carts id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.carts ALTER COLUMN id SET DEFAULT nextval('public.carts_id_seq'::regclass);


--
-- Name: items id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.items ALTER COLUMN id SET DEFAULT nextval('public.items_id_seq'::regclass);


--
-- Name: line_items id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.line_items ALTER COLUMN id SET DEFAULT nextval('public.line_items_id_seq'::regclass);


--
-- Name: orders id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.orders ALTER COLUMN id SET DEFAULT nextval('public.orders_id_seq'::regclass);


--
-- Name: refresh_tokens id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_tokens ALTER COLUMN id SET DEFAULT nextval('public.refresh_tokens_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: carts; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.carts (id, user_id, item_id, quantity) FROM stdin;
3	3	5	1
4	3	4	1
5	3	2	1
38	7	5	1
50	8	1	1
51	8	4	1
46	8	3	2
\.


--
-- Data for Name: items; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.items (id, name, description, price, stock, created_at) FROM stdin;
7	Notebook (A5)	Dot-grid, 200 pages, lay-flat binding.	1200	200	2026-08-06 14:13:17.245506
8	Cable Organiser	Leather magnetic cable ties, pack of 6.	800	148	2026-08-06 14:13:17.245506
1	Xsolla T-Shirt	Classic cotton tee with Xsolla logo. Unisex fit.	2500	114	2026-08-06 14:13:17.245506
6	USB-C Hub	7-in-1 hub: HDMI 4K, 3× USB-A, SD, microSD, PD.	3200	73	2026-08-06 14:13:17.245506
2	Developer Hoodie	Heavyweight pullover hoodie. Perfect for late-night PRs.	6000	43	2026-08-06 14:13:17.245506
4	Mechanical Keyboard	Tenkeyless, Cherry MX Brown switches. USB-C.	18000	16	2026-08-06 14:13:17.245506
9	Mechanical Keyboard	Compact mechanical keyboard with tactile switches	8900	0	2026-08-24 11:06:39.981634
5	Laptop Stand	Aluminium adjustable stand. Folds flat for travel.	4500	58	2026-08-06 14:13:17.245506
3	Sticker Pack	10-pack of Xsolla and open-source themed stickers.	500	282	2026-08-06 14:13:17.245506
10	Test Item 1	Description for test item 1	600	11	2026-08-29 09:10:56.379792
11	Test Item 2	Description for test item 2	700	12	2026-08-29 09:10:56.379792
12	Test Item 3	Description for test item 3	800	13	2026-08-29 09:10:56.379792
13	Test Item 4	Description for test item 4	900	14	2026-08-29 09:10:56.379792
14	Test Item 5	Description for test item 5	1000	15	2026-08-29 09:10:56.379792
15	Test Item 6	Description for test item 6	1100	16	2026-08-29 09:10:56.379792
16	Test Item 7	Description for test item 7	1200	17	2026-08-29 09:10:56.379792
17	Test Item 8	Description for test item 8	1300	18	2026-08-29 09:10:56.379792
18	Test Item 9	Description for test item 9	1400	19	2026-08-29 09:10:56.379792
19	Test Item 10	Description for test item 10	1500	20	2026-08-29 09:10:56.379792
20	Test Item 11	Description for test item 11	1600	21	2026-08-29 09:10:56.379792
21	Test Item 12	Description for test item 12	1700	22	2026-08-29 09:10:56.379792
22	Test Item 13	Description for test item 13	1800	23	2026-08-29 09:10:56.379792
23	Test Item 14	Description for test item 14	1900	24	2026-08-29 09:10:56.379792
24	Test Item 15	Description for test item 15	2000	25	2026-08-29 09:10:56.379792
25	Test Item 16	Description for test item 16	2100	26	2026-08-29 09:10:56.379792
26	Test Item 17	Description for test item 17	2200	27	2026-08-29 09:10:56.379792
27	Test Item 18	Description for test item 18	2300	28	2026-08-29 09:10:56.379792
28	Test Item 19	Description for test item 19	2400	29	2026-08-29 09:10:56.379792
29	Test Item 20	Description for test item 20	2500	30	2026-08-29 09:10:56.379792
30	Test Item 21	Description for test item 21	2600	31	2026-08-29 09:10:56.379792
31	Test Item 22	Description for test item 22	2700	32	2026-08-29 09:10:56.379792
32	Test Item 23	Description for test item 23	2800	33	2026-08-29 09:10:56.379792
33	Test Item 24	Description for test item 24	2900	34	2026-08-29 09:10:56.379792
34	Test Item 25	Description for test item 25	3000	35	2026-08-29 09:10:56.379792
35	Test Item 26	Description for test item 26	3100	36	2026-08-29 09:10:56.379792
36	Test Item 27	Description for test item 27	3200	37	2026-08-29 09:10:56.379792
37	Test Item 28	Description for test item 28	3300	38	2026-08-29 09:10:56.379792
38	Test Item 29	Description for test item 29	3400	39	2026-08-29 09:10:56.379792
39	Test Item 30	Description for test item 30	3500	40	2026-08-29 09:10:56.379792
40	Test Item 31	Description for test item 31	3600	41	2026-08-29 09:10:56.379792
41	Test Item 32	Description for test item 32	3700	42	2026-08-29 09:10:56.379792
42	Test Item 33	Description for test item 33	3800	43	2026-08-29 09:10:56.379792
43	Test Item 34	Description for test item 34	3900	44	2026-08-29 09:10:56.379792
44	Test Item 35	Description for test item 35	4000	45	2026-08-29 09:10:56.379792
45	Test Item 36	Description for test item 36	4100	46	2026-08-29 09:10:56.379792
46	Test Item 37	Description for test item 37	4200	47	2026-08-29 09:10:56.379792
47	Test Item 38	Description for test item 38	4300	48	2026-08-29 09:10:56.379792
48	Test Item 39	Description for test item 39	4400	49	2026-08-29 09:10:56.379792
49	Test Item 40	Description for test item 40	4500	50	2026-08-29 09:10:56.379792
50	Test Item 41	Description for test item 41	4600	51	2026-08-29 09:10:56.379792
51	Test Item 42	Description for test item 42	4700	52	2026-08-29 09:10:56.379792
52	Test Item 43	Description for test item 43	4800	53	2026-08-29 09:10:56.379792
53	Test Item 44	Description for test item 44	4900	54	2026-08-29 09:10:56.379792
54	Test Item 45	Description for test item 45	5000	55	2026-08-29 09:10:56.379792
55	Test Item 46	Description for test item 46	5100	56	2026-08-29 09:10:56.379792
56	Test Item 47	Description for test item 47	5200	57	2026-08-29 09:10:56.379792
57	Test Item 48	Description for test item 48	5300	58	2026-08-29 09:10:56.379792
58	Test Item 49	Description for test item 49	5400	59	2026-08-29 09:10:56.379792
59	Test Item 50	Description for test item 50	5500	60	2026-08-29 09:10:56.379792
\.


--
-- Data for Name: line_items; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.line_items (id, order_id, item_id, price, quantity) FROM stdin;
1	33	1	2500	2
2	33	3	500	1
3	34	1	2500	2
4	34	3	500	1
5	35	2	6000	1
6	35	4	18000	1
7	35	5	4500	1
8	36	8	800	1
9	36	6	3200	1
10	37	8	800	1
11	38	3	500	1
12	39	3	500	1
13	39	1	2500	1
14	40	3	500	1
15	40	1	2500	1
16	40	6	3200	1
17	40	2	6000	1
18	40	4	18000	1
19	41	5	4500	1
20	42	3	500	1
21	43	3	500	1
22	44	3	500	2
23	45	3	500	2
24	46	3	500	2
25	47	3	500	3
26	48	3	500	2
\.


--
-- Data for Name: orders; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.orders (id, user_id, total, status, created_at) FROM stdin;
1	1	3000	paid	2026-08-06 14:13:17.245506
2	2	18000	pending	2026-08-06 14:13:17.245506
3	1	9200	failed	2026-08-06 14:13:17.245506
4	3	3600	paid	2026-08-06 14:13:17.245506
5	2	6100	paid	2026-08-06 14:13:17.245506
6	1	9200	pending	2026-08-06 14:13:17.245506
7	3	22500	paid	2026-08-06 14:13:17.245506
8	1	2500	paid	2026-08-06 14:13:17.245506
9	2	24000	failed	2026-08-06 14:13:17.245506
10	3	4200	paid	2026-08-06 14:13:17.245506
11	1	6400	paid	2026-08-06 14:13:17.245506
12	3	6800	pending	2026-08-06 14:13:17.245506
13	2	7500	paid	2026-08-06 14:13:17.245506
14	1	6200	paid	2026-08-06 14:13:17.245506
15	3	18000	failed	2026-08-06 14:13:17.245506
16	2	7000	paid	2026-08-06 14:13:17.245506
17	1	5200	paid	2026-08-06 14:13:17.245506
18	3	7000	pending	2026-08-06 14:13:17.245506
19	2	4800	paid	2026-08-06 14:13:17.245506
20	1	18000	paid	2026-08-06 14:13:17.245506
21	3	9600	paid	2026-08-06 14:13:17.245506
22	1	10500	failed	2026-08-06 14:13:17.245506
23	2	3800	paid	2026-08-06 14:13:17.245506
24	3	18000	pending	2026-08-06 14:13:17.245506
25	1	12000	paid	2026-08-06 14:13:17.245506
26	2	6500	paid	2026-08-06 14:13:17.245506
27	3	5600	failed	2026-08-06 14:13:17.245506
28	1	3800	paid	2026-08-06 14:13:17.245506
29	2	9200	pending	2026-08-06 14:13:17.245506
30	3	19300	paid	2026-08-06 14:13:17.245506
31	1	6900	paid	2026-08-06 14:13:17.245506
32	2	4900	paid	2026-08-06 14:13:17.245506
33	1	5500	paid	2026-08-06 19:23:50.212229
34	2	5500	paid	2026-08-07 10:08:49.013517
35	7	28500	paid	2026-08-22 10:32:08.300441
36	7	4000	paid	2026-08-23 07:19:18.229617
37	7	800	paid	2026-08-23 08:57:23.252466
38	7	500	paid	2026-08-23 09:05:18.403893
39	7	3000	failed	2026-08-24 06:37:56.095664
40	7	30200	failed	2026-08-24 09:17:12.63473
41	8	4500	failed	2026-08-24 11:42:35.907926
42	8	500	failed	2026-08-24 11:42:49.136486
43	8	500	failed	2026-08-24 11:42:52.387581
44	8	1000	failed	2026-08-24 11:42:56.945358
45	8	1000	failed	2026-08-24 11:42:57.431974
46	8	1000	failed	2026-08-24 11:44:24.461979
47	8	1500	failed	2026-08-24 11:46:04.423975
48	8	1000	paid	2026-08-24 11:49:47.754589
\.


--
-- Data for Name: refresh_tokens; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.refresh_tokens (id, user_id, token_hash, expires_at, is_active, created_at) FROM stdin;
1	1	\\x12eb22e4741fd0f19bb667af3396a1da96dd98a802d11b88f081246a7b4c8aa7	2026-09-06 00:40:51.548408	t	2026-08-06 20:40:51.620428
2	1	\\x3495024a3e054af80b22764328b27df4d57dea686bcaad248094183ad8df7190	2026-09-06 00:50:25.155476	f	2026-08-06 20:50:25.234966
3	1	\\x0028709da2cdbdf7d6c6d99fc513160d5a7532c19ce6f0f49d233e92c0e34715	2026-09-06 00:50:57.961662	t	2026-08-06 20:50:57.967341
4	2	\\x8cce48ec6ba5d318702e4264394516871875a8f7720a85f7ed0895faa5bf3a58	2026-09-06 14:07:55.358496	f	2026-08-07 10:07:54.386042
5	2	\\xba2f873290c91d9d10b21332578febc31f5eecce6d42bb79b03a9ba966bddb62	2026-09-06 14:09:13.133059	t	2026-08-07 10:09:13.919136
6	3	\\x555d4b1fc558bc51a97bbaa96c758c696d3b950d1e44eac94d992bf14a87f7e7	2026-09-19 13:20:30.48687	t	2026-08-20 09:20:30.462929
7	3	\\x92a9c0b4b80a2dc457d66ac253b080bc3cc2a59ae1ea7a92bc88a6047090d091	2026-09-19 13:24:15.531051	t	2026-08-20 09:24:15.570122
8	3	\\x26b281d401578336491e97c1078d5105e94e559b08acdcf030cc0273bca2eea7	2026-09-21 12:49:01.658386	t	2026-08-22 08:49:01.646636
9	3	\\xe2c5e02f1fc48a39df0c6ab94830a0aed8f14b81202af40bac7730825fc85275	2026-09-21 12:49:14.147221	t	2026-08-22 08:49:14.087486
10	3	\\xc45cf9a4e7b12b895b6749b7725a5d5651645d6346a3b7f94b332482a66cbde6	2026-09-21 13:02:07.321769	t	2026-08-22 09:02:07.367315
11	7	\\x1f6ce5d963f8220a8f6966ac64d56a35eaece6d4efec60f2fe268152922f3fe3	2026-09-21 13:42:23.070298	t	2026-08-22 09:42:23.078341
12	7	\\xcbaf1124404643d3b81ef90fdda5d8c2770159bc231f4824e7298b70a6dff4f0	2026-09-21 13:43:39.319881	t	2026-08-22 09:43:39.278943
13	7	\\xe4b0aba580f83927018e74b6260a726548c1850984edca43e2ac11e0553c96e7	2026-09-21 14:32:05.893495	t	2026-08-22 10:32:05.842556
14	7	\\x5940a17ed64572d42166e1d3ed4035109610aae4dced5d914910e3a7bafacf25	2026-09-21 20:19:57.721438	t	2026-08-22 16:19:57.621286
15	7	\\x52a3fe70fcca9519984b69e5c62e00165b8b731399a2912f0f1d3baee2d676af	2026-09-22 10:49:24.343837	t	2026-08-23 06:49:23.456094
16	7	\\x05603739b2211f4bed63571c763c4c0c11503439d0cb1edd5e3836cc3224e1c6	2026-09-22 11:18:48.203207	t	2026-08-23 07:18:48.348435
17	7	\\x5d6399fbde8efdb69064711bed444f571158c3039b69dbf226f4ba429da17d48	2026-09-22 12:44:39.156504	t	2026-08-23 08:44:38.828537
18	7	\\xd326acf949b882ab8f41d78f699549c02e7e34296fab7372a52e90469e279131	2026-09-22 12:57:15.139449	t	2026-08-23 08:57:15.400121
19	7	\\x7e683cb8d4995f697197739a14206b82e068bc4b655626ff8a6a056e5536495a	2026-09-22 13:05:09.002549	t	2026-08-23 09:05:09.465855
20	7	\\x216183c876f6abe7789d58528d9be97eaa2cdad098980c411108e4ab8a2467de	2026-09-22 20:56:50.081748	t	2026-08-23 16:56:50.178277
21	7	\\x2e1707fe085ddd15fc685931687935c26d26899b6f1fc4dbdcf22df010c9c167	2026-09-22 21:04:00.143235	t	2026-08-23 17:04:01.207852
22	7	\\x603f9cd1440e97557f0ec124bf7bd3f5169a8d5fafd89eb7cd619383ed7ceb36	2026-09-23 07:45:06.013766	t	2026-08-24 03:45:07.570928
23	7	\\x0bad8e6a52daefe5b3f1c0d4dd9042e0738408705306988bd60fd2a981c04103	2026-09-23 07:54:34.610113	t	2026-08-24 03:54:34.859103
24	7	\\xa351a157d7b6f21b88d25f01a65300a2c00e90115c5cb2600e002e226fc7a69b	2026-09-23 10:37:49.929038	t	2026-08-24 06:37:49.479056
25	7	\\xadacbbb261993a25d3180c11cf43ddfc9c52a7ae4faff03f3127ea4018938692	2026-09-23 10:49:10.269523	t	2026-08-24 06:49:11.880128
26	7	\\x013099482eef5e3220290238c31f299859897d06ba5e57295b63019fa4695e89	2026-09-23 11:13:38.508104	f	2026-08-24 07:13:39.681754
27	7	\\x3eb526b2986e4c3c0023f9ddaa0d4243a8491f62ba6a7c4224824a35c6f7cce2	2026-09-23 11:36:40.230503	t	2026-08-24 07:36:41.336941
28	7	\\x4f82bf1bfd5021c5a8b8078751ffc29d198ff31e11ec421d3057fad48a7cf090	2026-09-23 11:41:52.324404	t	2026-08-24 07:41:52.944539
29	7	\\x74fd99cc5a6d35ba7453698681aa089668d42a7c3168b9769d1fba84e0189207	2026-09-23 12:14:33.133399	f	2026-08-24 08:14:34.20058
30	7	\\x84feb71f26bab59f15a1543538b82b9e3853c299c87136ebd5a4e7dbc49040cf	2026-09-23 12:31:54.914139	t	2026-08-24 08:31:56.572114
31	7	\\x2dddb90051fffc855005628aa9d3265b58ac831c87f846b549c05664ad69136f	2026-09-23 12:32:16.941331	t	2026-08-24 08:32:16.803831
32	7	\\x782a55a184a0117cbf79a2c012e18c91f43588ff3438b402619f616720c2306b	2026-09-23 12:39:40.471065	t	2026-08-24 08:39:41.628833
33	7	\\xcbc21a9f6d7bfa2df519c608069d68da0f2b1cf1b8108fe44dee5cd8dde0e190	2026-09-23 13:17:01.991443	t	2026-08-24 09:17:02.393386
34	7	\\xd1d75816656debf03db9a129efda3ea79bd864388b1fc14829bfc6fcb60939a9	2026-09-23 14:14:42.193286	t	2026-08-24 10:14:42.690289
35	7	\\x09500da9069d66442823e3d36812aabd1f0e40ebc11aa2bf17dcceb9ab213546	2026-09-23 15:03:40.920361	t	2026-08-24 11:03:39.75407
36	8	\\xb27a4d0601b3adbb6ab84973e49131e3ef4c7832da3a57f2710139a98bde7a21	2026-09-23 15:41:47.147894	t	2026-08-24 11:41:47.150499
37	8	\\x08f6eb342074b731c96874d3aae5587e356736270b156a6bf461b3af522cccf4	2026-09-23 15:55:06.943908	t	2026-08-24 11:55:06.944339
38	8	\\xf95487a7c254203dd7734e1a5b3b3d35bacac034e1097a6a2b0ae9db9ae115b5	2026-09-23 16:02:01.255409	t	2026-08-24 12:02:01.273086
39	7	\\x53d0b75f076c274b94ee551a64522db0e6f6f5ce9db422d4802da72d16ce1828	2026-09-26 13:42:06.0038	t	2026-08-27 09:42:07.690881
40	7	\\x61ce77a32d9143ccac2eae63d17ba02871e7710811a630c5b4d756a8e2fce819	2026-09-26 21:17:48.530077	t	2026-08-27 17:17:49.306349
41	7	\\x8eea849d9b46ddc78a522506508afe869ac83976a19325908e7bf1cb05849be7	2026-09-28 13:18:54.857215	t	2026-08-29 09:18:54.879231
42	7	\\xce954c2907177ef634107cc6bfc1ee6974dee99fbb6544efe9b740247561cc3c	2026-09-28 13:25:18.712468	t	2026-08-29 09:25:18.721109
43	7	\\x306c30dff9421d89f4007bede72c552662d55280f418b6bc727cc9d76a0c3867	2026-09-28 13:26:35.109897	t	2026-08-29 09:26:35.125737
44	7	\\xcba9f403f0518d9c597e8b932400be2b76d60303c9c25e57031f240408db86dc	2026-09-29 14:28:29.338796	t	2026-08-30 10:28:29.039897
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.schema_migrations (version, dirty) FROM stdin;
11	f
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, email, hash, created_at, updated_at) FROM stdin;
1	faridababy@example.com	\\x243261243130242f74574c4c726f30566776416950576a556857556c75744b7063667245386469796e42524e4f6a53774e566a57524c59542e6f7979	2026-08-06 15:41:55.539222	2026-08-06 15:41:55.539222
2	farid@example.com	\\x2432612431302432487a52725877517977424552365a6865334977612e696a7963416f66455866394a636939584f656d317870385573443055434657	2026-08-07 10:07:36.386836	2026-08-07 10:07:36.386836
3	sfarida@example.com	\\x2432612431302458627050776676324e783741634c4e615a4b716d72755430325278646a7770387a367278513739324f426c772e455578462e37412e	2026-08-20 07:57:42.684939	2026-08-20 07:57:42.684939
5	omar@example.com	\\x24326124313024726d36522f5267437676746d58435a6c6a39506d55656d7063514371426271325a64736234676e636d497a7356724f784a36785032	2026-08-22 09:13:30.385398	2026-08-22 09:13:30.385398
7	bamiye@example.com	\\x243261243130244c46693177436f707a6551665051464a776b6347642e33556c7171325a39533078656c2f7a64574177445674504c596b4957355279	2026-08-22 09:21:07.789575	2026-08-22 09:21:07.789575
8	hi@example.com	\\x24326124313024716e42783352476f5742656253495976694c554a3265455a3172677963386e4b61747a6351344f46334c334968596c5a4230375036	2026-08-24 11:41:26.686201	2026-08-24 11:41:26.686201
\.


--
-- Name: carts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.carts_id_seq', 56, true);


--
-- Name: items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.items_id_seq', 59, true);


--
-- Name: line_items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.line_items_id_seq', 26, true);


--
-- Name: orders_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.orders_id_seq', 48, true);


--
-- Name: refresh_tokens_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.refresh_tokens_id_seq', 44, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.users_id_seq', 8, true);


--
-- Name: carts carts_user_id_item_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.carts
    ADD CONSTRAINT carts_user_id_item_id_key UNIQUE (user_id, item_id);


--
-- Name: items items_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_pkey PRIMARY KEY (id);


--
-- Name: line_items line_items_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.line_items
    ADD CONSTRAINT line_items_pkey PRIMARY KEY (id);


--
-- Name: orders orders_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_email_hash; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_email_hash ON public.users USING btree (email, hash);


--
-- Name: idx_items_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_items_name ON public.items USING btree (name);


--
-- Name: idx_items_price; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_items_price ON public.items USING btree (price);


--
-- Name: idx_line_items_item_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_line_items_item_id ON public.line_items USING btree (item_id);


--
-- Name: idx_line_items_order_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_line_items_order_id ON public.line_items USING btree (order_id);


--
-- Name: idx_orders_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_orders_created_at ON public.orders USING btree (created_at DESC);


--
-- Name: idx_orders_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_orders_status ON public.orders USING btree (status);


--
-- Name: idx_orders_user_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_orders_user_id ON public.orders USING btree (user_id);


--
-- Name: idx_refresh_tokens_expires_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_refresh_tokens_expires_at ON public.refresh_tokens USING btree (expires_at);


--
-- Name: idx_refresh_tokens_user_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_refresh_tokens_user_id ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_user_id_item_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_id_item_id ON public.carts USING btree (user_id, item_id);


--
-- Name: carts carts_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.carts
    ADD CONSTRAINT carts_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id);


--
-- Name: line_items line_items_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.line_items
    ADD CONSTRAINT line_items_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id);


--
-- Name: line_items line_items_order_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.line_items
    ADD CONSTRAINT line_items_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict bSDNAUiFzfPMN6W1Lv0chFtfbjCssWTW4dGIUBYLUSVQetfN2d1tAojGDUA5gf5

