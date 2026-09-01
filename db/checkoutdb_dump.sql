--
-- PostgreSQL database dump
--

\restrict nWIiNH9dzvef7sZIMlsW8CAFqO33ft6do3n25aAlFCZ6tEcEi8XnRotqsmpX3iU

-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

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
-- Name: cart_items; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.cart_items (
    id integer NOT NULL,
    cart_id integer NOT NULL,
    item_id integer NOT NULL,
    quantity integer NOT NULL,
    CONSTRAINT cart_items_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.cart_items OWNER TO lala;

--
-- Name: cart_items_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.cart_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.cart_items_id_seq OWNER TO lala;

--
-- Name: cart_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.cart_items_id_seq OWNED BY public.cart_items.id;


--
-- Name: carts; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.carts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    last_accessed timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.carts OWNER TO lala;

--
-- Name: carts_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.carts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.carts_id_seq OWNER TO lala;

--
-- Name: carts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.carts_id_seq OWNED BY public.carts.id;


--
-- Name: idempotency_records; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.idempotency_records (
    key text NOT NULL,
    request_hash character varying NOT NULL,
    status_code integer NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.idempotency_records OWNER TO lala;

--
-- Name: items; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.items (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    price integer NOT NULL,
    stock integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.items OWNER TO lala;

--
-- Name: items_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.items_id_seq OWNER TO lala;

--
-- Name: items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.items_id_seq OWNED BY public.items.id;


--
-- Name: order_items; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.order_items (
    id integer NOT NULL,
    order_id integer NOT NULL,
    item_id integer NOT NULL,
    price integer NOT NULL,
    quantity integer NOT NULL,
    CONSTRAINT order_items_price_check CHECK ((price >= 0)),
    CONSTRAINT order_items_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.order_items OWNER TO lala;

--
-- Name: order_items_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.order_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.order_items_id_seq OWNER TO lala;

--
-- Name: order_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.order_items_id_seq OWNED BY public.order_items.id;


--
-- Name: orders; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.orders (
    id integer NOT NULL,
    user_id integer NOT NULL,
    total integer NOT NULL,
    status character varying(20) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.orders OWNER TO lala;

--
-- Name: orders_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.orders_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.orders_id_seq OWNER TO lala;

--
-- Name: orders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.orders_id_seq OWNED BY public.orders.id;


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.refresh_tokens (
    token text NOT NULL,
    user_id integer NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.refresh_tokens OWNER TO lala;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO lala;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.sessions (
    token text NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.sessions OWNER TO lala;

--
-- Name: users; Type: TABLE; Schema: public; Owner: lala
--

CREATE TABLE public.users (
    id integer NOT NULL,
    username character varying(50) NOT NULL,
    password_hash text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.users OWNER TO lala;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: lala
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO lala;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lala
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: cart_items id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.cart_items ALTER COLUMN id SET DEFAULT nextval('public.cart_items_id_seq'::regclass);


--
-- Name: carts id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.carts ALTER COLUMN id SET DEFAULT nextval('public.carts_id_seq'::regclass);


--
-- Name: items id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.items ALTER COLUMN id SET DEFAULT nextval('public.items_id_seq'::regclass);


--
-- Name: order_items id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.order_items ALTER COLUMN id SET DEFAULT nextval('public.order_items_id_seq'::regclass);


--
-- Name: orders id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.orders ALTER COLUMN id SET DEFAULT nextval('public.orders_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: cart_items; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.cart_items (id, cart_id, item_id, quantity) FROM stdin;
1	1	1	2
2	1	2	1
3	2	1	1
4	2	2	3
5	2	3	1
6	3	2	100
7	4	5	1
123	59	1	3
124	59	2	4
125	59	3	3
126	59	5	1
127	59	8	1
18	9	5	1
17	9	2	1
40	9	3	1
118	56	1	3
119	56	2	4
120	56	3	3
121	56	5	1
122	56	8	1
\.


--
-- Data for Name: carts; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.carts (id, user_id, last_accessed) FROM stdin;
1	1	2026-08-10 00:44:00.677669
2	2	2026-08-10 00:48:18.757133
3	3	2026-08-10 00:49:10.633212
4	4	2026-08-10 00:49:44.130612
59	13	2026-08-31 14:09:26.73697
9	0	2026-08-24 00:25:32.697569
56	11	2026-08-31 14:04:31.122059
\.


--
-- Data for Name: idempotency_records; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.idempotency_records (key, request_hash, status_code, body, created_at) FROM stdin;
55070bdb-6ff6-4093-bef1-a9613750d072	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":1,"user_id":5,"items":[{"item_id":5,"quantity":1,"price":4500}],"total":4500,"status":"paid"},"payment":{"success":true}}	2026-08-23 21:18:10.776039+04
788b5ab5-38b1-461d-9f48-3462abf79f21	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":2,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":9000,"status":"paid"},"payment":{"success":true}}	2026-08-23 22:34:48.604101+04
dd8092a2-ddf6-4ca3-b0cc-d4662d94524c	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":3,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":9000,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:06:26.48754+04
70ae13a2-e39a-4447-84e0-bd88b45088d0	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":4,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:06:33.489826+04
1d4fbd4d-dc9d-4df8-9d4e-5fc8f759d480	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":5,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:24:42.748433+04
7d4166c2-f6f2-4eef-9de3-fb014dfc178d	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":6,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:25:31.353328+04
8c804353-e478-4b66-a346-4b8f2b4a73c6	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":7,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:26:15.263778+04
d5306b94-0c04-4274-88bb-e66c30e18f17	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":8,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:27:30.35878+04
53b80bca-e283-4533-9216-1505cfd7c2d2	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":9,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:27:58.912044+04
dd7a9a3e-52dc-4510-b5c1-e4dfaf8e70e2	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":10,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:28:39.763153+04
1f0124a1-b2f5-43de-af5e-ece18d5edb78	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":11,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000}],"total":6000,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:29:41.393434+04
02a9fc69-23a5-45dd-9915-a76da73916de	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":12,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:33:32.16035+04
0d321359-f77f-4948-b473-f27809b218fc	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":13,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000}],"total":6000,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:35:08.046974+04
a9bbd900-6704-45c0-9ff2-f98f7dabcaaf	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":14,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000}],"total":6000,"status":"paid"},"payment":{"success":true}}	2026-08-23 23:35:46.020258+04
e82eef19-cca2-45e6-b6c9-afc658a6f6d0	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":15,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500}],"total":2500,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:16:38.608272+04
20afc6d1-e86b-43ba-9b1f-a48325fb01e8	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":16,"user_id":5,"items":[{"item_id":1,"quantity":1,"price":2500},{"item_id":2,"quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:21:56.355339+04
aa98bbb1-01d5-45e7-b72d-cd546a0b4f92	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":17,"user_id":5,"items":[{"item_id":2,"quantity":12,"price":6000}],"total":72000,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:27:08.820068+04
0b98c8ad-33d6-485d-961f-40f951b8a436	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":18,"user_id":5,"items":[{"item_id":5,"quantity":1,"price":4500},{"item_id":6,"quantity":1,"price":3200},{"item_id":8,"quantity":1,"price":800},{"item_id":9,"quantity":1,"price":120000}],"total":128500,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:27:39.320294+04
1696b07b-74d1-4a83-8423-6920c1bb1197	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":19,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:44:19.301702+04
a90cb20a-a5d0-4250-a5c5-7a6179fa94d1	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":20,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-24 00:58:50.427041+04
c0caf9c6-2746-41b0-9a5f-4b5abb0016f0	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":21,"user_id":5,"items":[{"item_id":2,"quantity":1,"price":6000},{"item_id":3,"quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-24 01:07:42.09198+04
eba12c2a-21fd-4183-8feb-06c6225559bc	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":22,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 01:44:04.94681+04
3597b46a-c33c-4c46-a9d2-506eb2d765b0	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":23,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 01:44:19.608457+04
02c8fa51-87a2-496c-8491-f2e531d184e4	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":24,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 01:48:02.665041+04
7c088d66-2e03-4d23-a091-f3d9f4971630	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":25,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500}],"total":2500,"status":"paid"},"payment":{"success":true}}	2026-08-24 02:08:54.667719+04
32d96441-6c83-4f18-994f-af56e4cdb385	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":26,"user_id":10,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":9,"name":"","quantity":1,"price":120000}],"total":122500,"status":"paid"},"payment":{"success":true}}	2026-08-24 02:31:50.666284+04
6bd545df-91fe-4c8d-b23a-9829727e3761	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":27,"user_id":5,"items":[{"item_id":2,"name":"","quantity":1,"price":6000},{"item_id":3,"name":"","quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-24 02:38:07.735203+04
61f1803c-1c19-4d6a-849a-d52b31b8e30f	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":28,"user_id":5,"items":[{"item_id":2,"name":"","quantity":1,"price":6000},{"item_id":3,"name":"","quantity":1,"price":500}],"total":6500,"status":"paid"},"payment":{"success":true}}	2026-08-24 02:52:29.40011+04
a7438b88-742b-4d37-b1be-c3a08b08cd7d	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":29,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 02:59:46.289855+04
183f537d-b75b-469d-8a0c-2cfc224a3aa4	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":30,"user_id":5,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":4,"name":"","quantity":1,"price":18000},{"item_id":6,"name":"","quantity":1,"price":3200}],"total":23700,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:00:18.742151+04
789af580-d8b3-4c9e-8f9b-dd216c530e77	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":31,"user_id":10,"items":[{"item_id":4,"name":"","quantity":1,"price":18000},{"item_id":9,"name":"","quantity":1,"price":120000}],"total":138000,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:44:58.018983+04
8249c07a-86b1-46c1-8444-c8e04433d2b2	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":32,"user_id":10,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":8500,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:48:00.814347+04
3473b8fa-5be8-4a89-911e-3434a06f13b2	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":33,"user_id":10,"items":[{"item_id":1,"name":"","quantity":1,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000},{"item_id":3,"name":"","quantity":1,"price":500}],"total":9000,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:50:32.942586+04
90f6ca2a-7fc3-435e-b97c-34c10cc6cfcc	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":34,"user_id":10,"items":[{"item_id":6,"name":"","quantity":1,"price":3200},{"item_id":9,"name":"","quantity":1,"price":120000}],"total":123200,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:51:14.008774+04
7fe25f50-dc17-409b-a6d0-a430851eea50	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":35,"user_id":10,"items":[{"item_id":2,"name":"","quantity":3,"price":6000},{"item_id":4,"name":"","quantity":3,"price":18000},{"item_id":5,"name":"","quantity":1,"price":4500},{"item_id":7,"name":"","quantity":1,"price":1200}],"total":77700,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:51:36.129969+04
914753aa-d6d2-4d2b-8a90-fb3b867baa00	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":36,"user_id":10,"items":[{"item_id":2,"name":"","quantity":5,"price":6000},{"item_id":5,"name":"","quantity":7,"price":4500}],"total":61500,"status":"paid"},"payment":{"success":true}}	2026-08-24 03:51:50.292426+04
5dee771a-6fd7-4921-816d-85e797ce7e22	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":37,"user_id":10,"items":[{"item_id":2,"name":"","quantity":2,"price":6000},{"item_id":3,"name":"","quantity":2,"price":500}],"total":13000,"status":"paid"},"payment":{"success":true}}	2026-08-24 09:49:06.363385+04
66af274b-542d-4d04-931b-3b02ed15487a	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":38,"user_id":10,"items":[{"item_id":1,"name":"","quantity":3,"price":2500},{"item_id":2,"name":"","quantity":1,"price":6000}],"total":13500,"status":"paid"},"payment":{"success":true}}	2026-08-24 10:09:59.399784+04
17774a72-be91-40ca-80b8-bc87f544366d	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":39,"user_id":10,"items":[{"item_id":1,"name":"","quantity":1,"price":2500}],"total":2500,"status":"paid"},"payment":{"success":true}}	2026-08-24 10:12:15.867906+04
a582b845-55d3-4584-8f01-86bd87b3f991	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":40,"user_id":10,"items":[{"item_id":3,"name":"","quantity":3,"price":500},{"item_id":4,"name":"","quantity":1,"price":18000}],"total":19500,"status":"paid"},"payment":{"success":true}}	2026-08-24 15:01:17.146871+04
e875852b-efb5-4355-bd07-97c76b460044	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":41,"user_id":10,"items":[{"item_id":1,"name":"","quantity":2,"price":2500},{"item_id":2,"name":"","quantity":2,"price":6000}],"total":17000,"status":"paid"},"payment":{"success":true}}	2026-08-24 15:27:03.578003+04
72492e8a-47aa-4a4f-9c31-9896511768d9	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":42,"user_id":10,"items":[{"item_id":6,"name":"","quantity":1,"price":3200},{"item_id":8,"name":"","quantity":1,"price":800}],"total":4000,"status":"paid"},"payment":{"success":true}}	2026-08-24 15:28:00.798671+04
771b0154-a91e-4b46-8631-884bec259d5d	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":43,"user_id":10,"items":[{"item_id":4,"name":"","quantity":1,"price":18000},{"item_id":5,"name":"","quantity":1,"price":4500}],"total":22500,"status":"paid"},"payment":{"success":true}}	2026-08-24 15:28:26.635741+04
c92608bd-9d20-4b7b-b377-2ad4e678eb21	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":44,"user_id":10,"items":[{"item_id":2,"name":"","quantity":1,"price":6000}],"total":6000,"status":"paid"},"payment":{"success":true}}	2026-08-25 09:55:52.881322+04
7c1c0620-1d29-4687-a704-037c2c01b76c	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":45,"user_id":10,"items":[{"item_id":2,"name":"","quantity":4,"price":6000},{"item_id":3,"name":"","quantity":1,"price":500}],"total":24500,"status":"paid"},"payment":{"success":true}}	2026-08-25 10:10:39.879271+04
1977e1e6-def0-4cdb-95ab-55657228b573	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":46,"user_id":10,"items":[{"item_id":8,"name":"","quantity":1,"price":800},{"item_id":9,"name":"","quantity":1,"price":120000}],"total":120800,"status":"paid"},"payment":{"success":true}}	2026-08-25 10:11:31.872929+04
a4d62096-ef69-48a3-bc51-be5400c56819	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":47,"user_id":10,"items":[{"item_id":2,"name":"","quantity":1,"price":6000},{"item_id":6,"name":"","quantity":1,"price":3200}],"total":9200,"status":"paid"},"payment":{"success":true}}	2026-08-25 10:15:00.40349+04
2	e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855	201	{"order":{"id":49,"user_id":15,"items":[{"item_id":4,"name":"Mechanical Keyboard","quantity":4,"price":18000},{"item_id":5,"name":"Laptop Stand","quantity":1,"price":4500}],"total":76500,"status":"paid"},"payment":{"success":true}}	2026-09-01 12:06:21.629693+04
62fa7173-4ee9-4454-b31d-dd3d714f17af	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":50,"user_id":15,"items":[{"item_id":1,"name":"Xsolla T-Shirt","quantity":3,"price":2500},{"item_id":2,"name":"Developer Hoodie","quantity":5,"price":6000}],"total":37500,"status":"paid"},"payment":{"success":true}}	2026-09-01 12:34:29.430606+04
996f3399-7774-41cc-85e9-08435bdb783f	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":51,"user_id":15,"items":[{"item_id":7,"name":"Notebook (A5)","quantity":2,"price":1200},{"item_id":8,"name":"Cable Organiser","quantity":1,"price":800}],"total":3200,"status":"paid"},"payment":{"success":true}}	2026-09-01 12:35:44.38381+04
20070c28-c004-48f4-bfc4-cbb77d6ac5f1	44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a	201	{"order":{"id":52,"user_id":15,"items":[{"item_id":2,"name":"Developer Hoodie","quantity":1,"price":6000},{"item_id":3,"name":"Sticker Pack","quantity":3,"price":500}],"total":7500,"status":"paid"},"payment":{"success":true}}	2026-09-01 12:41:21.212254+04
\.


--
-- Data for Name: items; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.items (id, name, description, price, stock, created_at) FROM stdin;
1	Xsolla T-Shirt	Classic cotton tee with Xsolla logo. Unisex fit.	2500	120	2026-08-09 19:10:22.235104
2	Developer Hoodie	Heavyweight pullover hoodie. Perfect for late-night PRs.	6000	45	2026-08-09 19:10:22.235104
3	Sticker Pack	10-pack of Xsolla and open-source themed stickers.	500	300	2026-08-09 19:10:22.235104
4	Mechanical Keyboard	Tenkeyless, Cherry MX Brown switches. USB-C.	18000	18	2026-08-09 19:10:22.235104
5	Laptop Stand	Aluminium adjustable stand. Folds flat for travel.	4500	60	2026-08-09 19:10:22.235104
6	USB-C Hub	7-in-1 hub: HDMI 4K, 3Ã— USB-A, SD, microSD, PD.	3200	75	2026-08-09 19:10:22.235104
7	Notebook (A5)	Dot-grid, 200 pages, lay-flat binding.	1200	200	2026-08-09 19:10:22.235104
8	Cable Organiser	Leather magnetic cable ties, pack of 6.	800	150	2026-08-09 19:10:22.235104
9	Laptop	A fast laptop	120000	10	2026-08-09 22:45:31.412613
\.


--
-- Data for Name: order_items; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.order_items (id, order_id, item_id, price, quantity) FROM stdin;
1	1	1	2500	1
2	1	3	500	1
3	2	4	18000	1
4	3	2	6000	1
5	3	6	3200	1
6	4	7	1200	3
7	5	5	4500	1
8	5	8	800	2
9	6	2	6000	1
10	6	6	3200	1
11	7	4	18000	1
12	7	5	4500	1
13	8	3	500	5
14	9	4	18000	1
15	9	2	6000	1
16	10	1	2500	1
17	10	7	1200	1
18	10	3	500	1
19	11	6	3200	2
20	12	2	6000	1
21	12	8	800	1
22	13	1	2500	3
23	14	5	4500	1
24	14	7	1200	1
25	14	3	500	1
26	15	4	18000	1
27	16	2	6000	1
28	16	3	500	2
29	17	6	3200	1
30	17	8	800	1
31	17	7	1200	1
32	18	1	2500	1
33	18	5	4500	1
34	19	7	1200	4
35	20	4	18000	1
36	21	6	3200	3
37	22	2	6000	1
38	22	5	4500	1
39	23	1	2500	1
40	23	3	500	1
41	23	8	800	1
42	24	4	18000	1
43	25	2	6000	2
44	26	5	4500	1
45	26	3	500	4
46	27	6	3200	1
47	27	7	1200	2
48	28	3	500	1
49	28	8	800	1
50	28	1	2500	1
51	29	2	6000	1
52	29	6	3200	1
53	30	4	18000	1
54	30	8	800	1
55	30	3	500	1
56	31	5	4500	1
57	31	8	800	3
58	32	3	500	5
59	32	7	1200	2
60	1	5	4500	1
61	2	1	2500	1
62	2	2	6000	1
63	2	3	500	1
64	3	1	2500	1
65	3	2	6000	1
66	3	3	500	1
67	4	2	6000	1
68	4	3	500	1
69	5	1	2500	1
70	5	2	6000	1
71	6	2	6000	1
72	6	3	500	1
73	7	1	2500	1
74	7	2	6000	1
75	8	2	6000	1
76	8	3	500	1
77	9	2	6000	1
78	9	3	500	1
79	10	2	6000	1
80	10	3	500	1
81	11	2	6000	1
82	12	1	2500	1
83	12	2	6000	1
84	13	2	6000	1
85	14	2	6000	1
86	15	1	2500	1
87	16	1	2500	1
88	16	2	6000	1
89	17	2	6000	12
90	18	5	4500	1
91	18	6	3200	1
92	18	8	800	1
93	18	9	120000	1
94	19	2	6000	1
95	19	3	500	1
96	20	2	6000	1
97	20	3	500	1
98	21	2	6000	1
99	21	3	500	1
100	22	1	2500	1
101	22	2	6000	1
102	23	1	2500	1
103	23	2	6000	1
104	24	1	2500	1
105	24	2	6000	1
106	25	1	2500	1
107	26	1	2500	1
108	26	9	120000	1
109	27	2	6000	1
110	27	3	500	1
111	28	2	6000	1
112	28	3	500	1
113	29	1	2500	1
114	29	2	6000	1
115	30	1	2500	1
116	30	4	18000	1
117	30	6	3200	1
118	31	4	18000	1
119	31	9	120000	1
120	32	1	2500	1
121	32	2	6000	1
122	33	1	2500	1
123	33	2	6000	1
124	33	3	500	1
125	34	6	3200	1
126	34	9	120000	1
127	35	2	6000	3
128	35	4	18000	3
129	35	5	4500	1
130	35	7	1200	1
131	36	2	6000	5
132	36	5	4500	7
133	37	2	6000	2
134	37	3	500	2
135	38	1	2500	3
136	38	2	6000	1
137	39	1	2500	1
138	40	3	500	3
139	40	4	18000	1
140	41	1	2500	2
141	41	2	6000	2
142	42	6	3200	1
143	42	8	800	1
144	43	4	18000	1
145	43	5	4500	1
146	44	2	6000	1
147	45	2	6000	4
148	45	3	500	1
149	46	8	800	1
150	46	9	120000	1
151	47	2	6000	1
152	47	6	3200	1
153	48	2	6000	4
154	49	4	18000	4
155	49	5	4500	1
156	50	1	2500	3
157	50	2	6000	5
158	51	7	1200	2
159	51	8	800	1
160	52	2	6000	1
161	52	3	500	3
\.


--
-- Data for Name: orders; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.orders (id, user_id, total, status, created_at) FROM stdin;
1	5	4500	paid	2026-08-23 21:18:10.722538
2	5	9000	paid	2026-08-23 22:34:48.601861
3	5	9000	paid	2026-08-23 23:06:26.486186
4	5	6500	paid	2026-08-23 23:06:33.488075
5	5	8500	paid	2026-08-23 23:24:42.744535
6	5	6500	paid	2026-08-23 23:25:31.351158
7	5	8500	paid	2026-08-23 23:26:15.261537
8	5	6500	paid	2026-08-23 23:27:30.356836
9	5	6500	paid	2026-08-23 23:27:58.909384
10	5	6500	paid	2026-08-23 23:28:39.760527
11	5	6000	paid	2026-08-23 23:29:41.384329
12	5	8500	paid	2026-08-23 23:33:32.15862
13	5	6000	paid	2026-08-23 23:35:08.044844
14	5	6000	paid	2026-08-23 23:35:46.017994
15	5	2500	paid	2026-08-24 00:16:38.602815
16	5	8500	paid	2026-08-24 00:21:56.350928
17	5	72000	paid	2026-08-24 00:27:08.817326
18	5	128500	paid	2026-08-24 00:27:39.315366
19	5	6500	paid	2026-08-24 00:44:19.298687
20	5	6500	paid	2026-08-24 00:58:50.423042
21	5	6500	paid	2026-08-24 01:07:42.089695
22	5	8500	paid	2026-08-24 01:44:04.938025
23	5	8500	paid	2026-08-24 01:44:19.605022
24	5	8500	paid	2026-08-24 01:48:02.656177
25	5	2500	paid	2026-08-24 02:08:54.662673
26	10	122500	paid	2026-08-24 02:31:50.648764
27	5	6500	paid	2026-08-24 02:38:07.731413
28	5	6500	paid	2026-08-24 02:52:29.388831
29	5	8500	paid	2026-08-24 02:59:46.286497
30	5	23700	paid	2026-08-24 03:00:18.739797
31	10	138000	paid	2026-08-24 03:44:58.015241
32	10	8500	paid	2026-08-24 03:48:00.810158
33	10	9000	paid	2026-08-24 03:50:32.937677
34	10	123200	paid	2026-08-24 03:51:14.006757
35	10	77700	paid	2026-08-24 03:51:36.127845
36	10	61500	paid	2026-08-24 03:51:50.290734
37	10	13000	paid	2026-08-24 09:49:06.355887
38	10	13500	paid	2026-08-24 10:09:59.394623
39	10	2500	paid	2026-08-24 10:12:15.863129
40	10	19500	paid	2026-08-24 15:01:17.140625
41	10	17000	paid	2026-08-24 15:27:03.57154
42	10	4000	paid	2026-08-24 15:28:00.79494
43	10	22500	paid	2026-08-24 15:28:26.633878
44	10	6000	paid	2026-08-25 09:55:52.871733
45	10	24500	paid	2026-08-25 10:10:39.875888
46	10	120800	paid	2026-08-25 10:11:31.86893
47	10	9200	paid	2026-08-25 10:15:00.39761
48	11	24000	paid	2026-08-31 02:22:36.272924
49	15	76500	paid	2026-09-01 12:06:21.62151
50	15	37500	paid	2026-09-01 12:34:29.42807
51	15	3200	paid	2026-09-01 12:35:44.381084
52	15	7500	paid	2026-09-01 12:41:21.209701
\.


--
-- Data for Name: refresh_tokens; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.refresh_tokens (token, user_id, expires_at, revoked, created_at) FROM stdin;
cd5c09d289e6b87cabec7e6620296a0afde5e1c80ccbc99c36773a3f38cf097d	5	2026-09-09 12:57:59.587767+04	f	2026-08-10 12:57:59.588989+04
55a4529dad40bb666aadc3c8afcc358646d6a07a00eeb02d49ab74e7cf631fe5	7	2026-09-09 13:53:32.058207+04	f	2026-08-10 13:53:32.05948+04
b0d3afe00c88e2056e1a76e21baf95cecd59496c6015bee1025871ec9ece16e3	7	2026-09-09 13:56:16.333804+04	f	2026-08-10 13:56:16.334515+04
74e6cc6d8b3ff0ae2987c652bee734d4dab75548402809ffe9ec3942d83468f0	7	2026-09-09 14:00:00.80465+04	f	2026-08-10 14:00:00.805666+04
d6117397aba6222ddee4f7579d5037f5dc11bd3b9768742bf26fb66f52dc1b3f	7	2026-09-09 14:39:51.110777+04	f	2026-08-10 14:39:51.112039+04
4cd6e53d6efbaf497b881a5f30154b684ad2d3413e5641a1e9ffdee56376bd80	5	2026-09-19 13:16:40.387223+04	f	2026-08-20 13:16:40.391604+04
1a4a1e66963940c9b3f21de65df6b6558805cc20d7fe7767b56adfb42e6ebaca	5	2026-09-19 13:22:50.034155+04	f	2026-08-20 13:22:50.036144+04
60e962b9f6b38f2706cdee82ba5287dacef603639a52bb09bb9e72d73e2e1a3f	5	2026-09-22 20:47:18.738271+04	f	2026-08-23 20:47:18.739314+04
dc9dfa8bc25f6387bde9335d56b6189601f42a002188f5a8c671bff5966a8541	5	2026-09-22 20:47:22.047146+04	f	2026-08-23 20:47:22.05062+04
33465d0d6e62140f10d574e858e8bce4d3599cdf8edb0acd7b385f0ada4551d3	5	2026-09-22 20:47:24.480539+04	f	2026-08-23 20:47:24.481793+04
af5e2703215a5995551502d56c955c23c851804e899132eb74fff46aaa1c4bd3	5	2026-09-22 20:47:26.063091+04	f	2026-08-23 20:47:26.071823+04
90d8252d2a4147ef767bc95f485137eb75e1459bc91762ef08e6422f9357b115	5	2026-09-22 21:18:10.680615+04	f	2026-08-23 21:18:10.685477+04
fcf93633b4d9a4c1e57e48407f6860ee9e750620c828d45811399c6a767d678c	5	2026-09-22 22:34:48.551752+04	f	2026-08-23 22:34:48.553605+04
657e8ac1ce89e53f6ce0691d1b5c32db660f6ef3e37894ff5937a0aad5b4d68b	5	2026-09-22 23:06:26.466013+04	f	2026-08-23 23:06:26.475505+04
70880695452e04864c1c91d0e0228c9456fb49b2d7eb12aa679a1f810e0210fd	5	2026-09-22 23:24:42.70836+04	f	2026-08-23 23:24:42.709046+04
f39137240cb45b16fe61a367d14d202a77d3f42f69ff4b2212b5dcada1fdfaa7	5	2026-09-22 23:25:31.335319+04	f	2026-08-23 23:25:31.335762+04
39b994f332d7764df1839221bef03bdc57ff066d315a7e6fbe816e2ef081fa58	5	2026-09-22 23:26:15.234599+04	f	2026-08-23 23:26:15.237344+04
cbbcafa8474fe9eb5488ec6e8d41f35035bf69cced0b53caea197e49c20f38cb	5	2026-09-22 23:27:30.342339+04	f	2026-08-23 23:27:30.343099+04
88fc8a5e86a370594a0eef46696061326cb4f65bf1d1fe32845412ca40f24490	5	2026-09-22 23:27:58.896422+04	f	2026-08-23 23:27:58.896935+04
07192b1a765243f1a229112d26c61a43a80210abfa51828e65ccfd0b33bb2cb0	5	2026-09-22 23:28:39.746265+04	f	2026-08-23 23:28:39.74708+04
18588bf3662edc0a404f1fd8f3765225f0a0b0258713cc573e697d2bc23a48e4	5	2026-09-22 23:29:41.364959+04	f	2026-08-23 23:29:41.365256+04
d38eb03d7b4cba69ff7db54cc4ca5f653d1bea9d465fb8f6ce4792f439b22c3e	5	2026-09-22 23:33:32.143961+04	f	2026-08-23 23:33:32.144677+04
50aaf536cc90802bdfa47e7fef968c5916c80f9fca6188c0dc86768277f33f0d	5	2026-09-22 23:35:08.028139+04	f	2026-08-23 23:35:08.031602+04
34f3e6a8de3f5fe285420472c6fdd1450ff5aa2838daf6d44b5c20c892ba2574	5	2026-09-22 23:35:45.996011+04	f	2026-08-23 23:35:45.996916+04
cf9c511686ba8586e8f56e1a3bc6be15740612a606b778cb3cd55c745e134f83	5	2026-09-23 00:16:38.571665+04	f	2026-08-24 00:16:38.572447+04
2e3dad773bfdaf08e9a955564cfe31efeca93f2a5747015c8176d06b416ccffe	5	2026-09-23 00:21:56.327333+04	f	2026-08-24 00:21:56.327975+04
e1293169f05148575fde2c0a8deb816e7bcb11671bc5a5bd2622e4117dfd25c0	5	2026-09-23 00:27:08.796614+04	f	2026-08-24 00:27:08.802283+04
4594e46cbf1cae136c8c6a7770444baf9494e45062850e1a34f2c38d1fa80aa2	5	2026-09-23 00:44:19.274436+04	f	2026-08-24 00:44:19.28513+04
7523ba3b0b13f28174852b1558c44a37d48f4f35876852db0f03909785f81783	5	2026-09-23 00:58:50.399645+04	f	2026-08-24 00:58:50.402539+04
31b78b93e98ea5468e325713342ddc66c4cd510fbd561ee0fab53c13c9855a47	5	2026-09-23 01:07:42.070786+04	f	2026-08-24 01:07:42.072907+04
d9904ca2fc25216e3e61ab3b7e91d19cf4ee58c0c8ca29d0c6f515b812c0410c	5	2026-09-23 01:42:50.254336+04	f	2026-08-24 01:42:50.255646+04
560889553cf012176a3783127dbf8e144edc3b92028d53f625ce7eccedae7b39	5	2026-09-23 01:44:19.586829+04	f	2026-08-24 01:44:19.588562+04
6875fe7e16faf94bd1cb88d982858f70f9b80a053d97300a82d21cfdb261e945	5	2026-09-23 01:48:02.600796+04	f	2026-08-24 01:48:02.602887+04
5429625ade904e65cdf7a37363d10cfca4fcf1c6c1c167ee49eff4ca2aab9bcf	5	2026-09-23 02:08:54.631472+04	f	2026-08-24 02:08:54.635338+04
8e0771f3e111814fffef9d87142bc7e77148b5944e586a0a2ea9ae54a429b5b1	10	2026-09-23 02:31:24.94185+04	f	2026-08-24 02:31:24.943279+04
d3ca02ac60cad16d6de44796b55fc3998b6c51b27989853ab5dd1d7afe510573	10	2026-09-23 02:31:50.581691+04	f	2026-08-24 02:31:50.587146+04
4dda117a7806a560a6d98795b8c7156f4088a20f3928e24fe448922e1aba164e	5	2026-09-23 02:38:07.711934+04	f	2026-08-24 02:38:07.714997+04
724a346c18c3bafad40b27a28f79a6bf986f0a338a0133f58754398719b9393d	5	2026-09-23 02:52:29.317549+04	f	2026-08-24 02:52:29.325105+04
19bbe7ce4275a395209e2253506377c8f265ad479aaf888b12c570a4cc9d3995	5	2026-09-23 02:59:26.668734+04	f	2026-08-24 02:59:26.672038+04
6bd15f3bb4e9da598d774352b6d3e154988bf931d700151b540919c359ece262	10	2026-09-23 03:44:57.98988+04	f	2026-08-24 03:44:57.9942+04
a6ca0c255de9b664b02b0ce54b38aeaf7509421d2c712ac988b607cd66d430c8	10	2026-09-23 03:48:00.78062+04	f	2026-08-24 03:48:00.78797+04
cb063270b09c339b9e07c0ee7588a06a907f9d5035574c2257b149bf95554a09	10	2026-09-23 03:50:32.907739+04	f	2026-08-24 03:50:32.912843+04
8f8a195d134caec4e693d2331573f24bc1330e9007b13951a64469176fa71c78	10	2026-09-23 09:49:06.140694+04	f	2026-08-24 09:49:06.158048+04
d9f8929eec4a8e83fbbed6f312b6f0ad5b96849b63c72be48725ff032b702ac8	10	2026-09-23 10:09:59.357957+04	f	2026-08-24 10:09:59.367147+04
0f6c113f05a1372d58e99ee6222ff885127d1f272b6b8afe8710d71a4870fc40	10	2026-09-23 10:12:15.827481+04	f	2026-08-24 10:12:15.828741+04
5bace42a8588b7f9809b1546cebc66887ee473236cd50d21663e60a6f7f71dfd	10	2026-09-23 15:01:17.100882+04	f	2026-08-24 15:01:17.102046+04
7bc08bce336b0b4502c615b00235b8ece82ae360198fbe9ccfc508743d6ad61c	10	2026-09-23 15:16:09.765534+04	f	2026-08-24 15:16:09.765998+04
6977089e0d23f1d4d889b1ba0f5a5fc5ca67e5194a75302a8d81a4e98fba51bc	10	2026-09-23 15:27:03.523862+04	f	2026-08-24 15:27:03.525389+04
a23632bd3cd4e81fa5b70dcae79eca6bffd7f3b5a6b9802dab9b59c009062529	10	2026-09-23 15:28:00.766265+04	f	2026-08-24 15:28:00.772063+04
9b89495f522a6482713f2f32980a542cc7ed18e73dcd4fb4251e855be5b4cdfb	10	2026-09-24 09:55:52.813745+04	f	2026-08-25 09:55:52.818065+04
ee3381c704f1216191472e06517f1745c89acec1b41b6715916611e3ffca7ccf	10	2026-09-24 10:10:39.853567+04	f	2026-08-25 10:10:39.861226+04
1dee3f4fe8e1846608fca9d667022a52b920f068ea7ce55fb8d4c55930470ab0	10	2026-09-24 10:15:00.359176+04	f	2026-08-25 10:15:00.360777+04
b663f50443c4d9c42ccef0f183a165d39890f80d4dca8b5a37e8f864c69642b7	10	2026-09-24 10:20:57.506818+04	f	2026-08-25 10:20:57.515065+04
2e3791dc9f4c35e0f1e3746536c337a10e5014c38530d1621d7765691f38b997	10	2026-09-30 02:10:30.52279+04	f	2026-08-31 02:10:30.526029+04
8eb1c3958f2a7b5dba6d479940db6f84fc3fff71934b05540a63365fb576b6c1	11	2026-09-30 02:16:02.839355+04	f	2026-08-31 02:16:02.840141+04
688e0731e2ebe787eaa3cdc58f0b33031a8237684be02867926cf41fa09a1dcc	11	2026-09-30 02:23:25.296664+04	f	2026-08-31 02:23:25.297768+04
d7b94979b6819f9615d74f0dd169925cea550108b3b00301c6087024f320ce85	11	2026-09-30 02:23:27.48273+04	f	2026-08-31 02:23:27.483593+04
93ab827d8771f7f1747b8b7f2c963da951867af69f43dd2f98e096e0c73ddeed	11	2026-09-30 14:04:27.107706+04	f	2026-08-31 14:04:27.111769+04
cd07ea16f437583f076d1bad62ceaf899a30c0563863f04882c4f2ee368ad1fe	11	2026-09-30 14:04:30.550753+04	f	2026-08-31 14:04:30.558043+04
78263696a64bd4cf589b67f7879b3096507eff916bdc91d72daf5eb4b806a767	11	2026-09-30 14:04:31.113473+04	f	2026-08-31 14:04:31.115899+04
9c7fc925b38908695d3d096bf6c27ab22692d5b431d5c7b9a8bcaebd348675ca	13	2026-09-30 14:08:58.406828+04	f	2026-08-31 14:08:58.407449+04
8ca7a012c545378d230f38465819be778818a518aa737338acd89706cb86f6c7	13	2026-09-30 14:09:26.725991+04	f	2026-08-31 14:09:26.728526+04
ed17de19725bdd28035082e91cecab84b78822ce1b3d03e1bc70202793e6f1dc	15	2026-10-01 12:02:03.267762+04	f	2026-09-01 12:02:03.268784+04
6d37216368778bb5ef26d28f4dbbcd39f7808d7d167952cec7d772959c4b31eb	15	2026-10-01 12:08:22.860961+04	f	2026-09-01 12:08:22.861312+04
e6ba07f65f27c50c0678ae1e23452dfd4f87f93dbd69feddb107d0aa7eaf261c	15	2026-10-01 12:11:41.827672+04	f	2026-09-01 12:11:41.831023+04
fa953325f29ebc06da9791142887451999790e064515fafcce6d3e58b2a86d5b	15	2026-10-01 12:12:43.900211+04	f	2026-09-01 12:12:43.900827+04
854be00337664d60b0b458793275e9ada83ddb9166f74a611ebaea979d7696cb	15	2026-10-01 12:14:30.711896+04	f	2026-09-01 12:14:30.714524+04
856b483aa3648cbe118f280e87d625d43e3a8f0982d2b495ad550f9f03e807c7	15	2026-10-01 12:14:57.0507+04	f	2026-09-01 12:14:57.051838+04
ee8b9f15fd4e8f842ff6ef09947975da2b0a53850d8ef475480ae6634fbae783	15	2026-10-01 12:19:50.334804+04	f	2026-09-01 12:19:50.335161+04
ba34841ee44a8232b2c2be3101c4f6203f7e72f6d6b07f0627a88a12ebadc3d5	15	2026-10-01 12:20:11.874127+04	f	2026-09-01 12:20:11.876141+04
a19111152e4ed59181d3d481c27602334d9b09045d9967d4981172aa1c44c5fa	15	2026-10-01 12:20:34.700771+04	f	2026-09-01 12:20:34.703396+04
9a731b888431190afbf130e59bccd4a73c0c982a090245432ee6efafda7075b7	15	2026-10-01 12:27:27.500916+04	f	2026-09-01 12:27:27.50372+04
49643b3e4d044f35be8bf3ce13b758e4b6c59777d5447eedec7a4989d1370d1f	15	2026-10-01 12:27:30.889095+04	f	2026-09-01 12:27:30.89047+04
580ef418283fa8cde51d51739335a6e52b7e39e0f86f2c28c6d10faaa9d2e573	15	2026-10-01 12:29:26.729433+04	f	2026-09-01 12:29:26.731147+04
0e0debd5e5edaad93708452829fcea6336e340de96cbd7daea6df3fdcae6eeea	15	2026-10-01 12:33:03.969771+04	f	2026-09-01 12:33:03.970077+04
18c0258330e0887bbe60bae9126e7e66dd33aa613ea7dd948552ae713bf5d1da	15	2026-10-01 12:34:29.408768+04	f	2026-09-01 12:34:29.409724+04
66d05cc8f95f3cbe0f4f9d87f99d0d12f542f4d16fa5d06b9df68065306f85a5	15	2026-10-01 12:41:21.175542+04	f	2026-09-01 12:41:21.177504+04
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.schema_migrations (version, dirty) FROM stdin;
0	f
\.


--
-- Data for Name: sessions; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.sessions (token, user_id, created_at) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: lala
--

COPY public.users (id, username, password_hash, created_at) FROM stdin;
1	webgirl	$2a$10$X1d8YgceARZ0Ah84GBzYTOGGgq91RC/geogYJyqDv8I3SO2VKjCG6	2026-08-10 03:50:47.314496+04
2	lala	$2a$10$sqart5qnpQPUvtAFYz0bLOy0iKyxRBnuXAqMixJQGLHanqN9pEUmy	2026-08-10 03:51:10.544713+04
3	connor	$2a$10$lpx/AQpepkCaI3F4bqMXsO26MxgVdZme1NTR39pwgeA53ty64E8SG	2026-08-10 03:51:26.254988+04
5	randomName	$2a$10$rPjkw2JbFuJLxex9WrYiWePK0JDGMSns75J2bOx8l0TEf/LCpBlDW	2026-08-10 11:34:53.292108+04
6	Aynur	$2a$10$/WXuxf2.N9d0sVxDYUN/4Og/J8QapqEUPF8rV14JKDA08nD4g4o02	2026-08-10 12:44:47.167317+04
7	LalaA	$2a$10$1qICRlh85/zjTvol2.NTduTPBORumv23prrppr0gN4IapwmXu7ybG	2026-08-10 13:53:19.002776+04
8	user1	$2a$10$mxOillg14tbSz.AXKZQXl.09LOxyntD84dy9xB9.qORFvitE5OZwq	2026-08-10 14:39:42.281668+04
9	Lalaa@gmail.com	$2a$10$87ryD6bc1GjeyN.MK1ibYuT1mSa5Ix3ZP4hYAaGHlKHV44wrPTRqe	2026-08-20 13:12:26.601946+04
10	newUser	$2a$10$oDcOWdbwoo3cnQRFI4avqubIj9ku0MakdB4hyxqoCeHeEHxXI8Ewy	2026-08-24 02:31:03.972205+04
11	fnsm	$2a$10$ZcUBLN.Gp.WwcQmBeaxUz.ix0gILrFzWLbSRqNl7ucl5TzyR60p.S	2026-08-31 02:15:45.105397+04
13	hellow	$2a$10$7oj/9WDUlsbgiEj7si3X5OIfVI47670YaFUBAqEDLfIhf65KYNX7O	2026-08-31 14:08:45.467944+04
15	aynur	$2a$10$4EUoT17KNQzM/Ty/r4M73.eQwFdtwnxEK3huqem4V0bf3ozzxBURS	2026-09-01 12:01:42.591941+04
\.


--
-- Name: cart_items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.cart_items_id_seq', 155, true);


--
-- Name: carts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.carts_id_seq', 73, true);


--
-- Name: items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.items_id_seq', 9, true);


--
-- Name: order_items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.order_items_id_seq', 161, true);


--
-- Name: orders_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.orders_id_seq', 52, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: lala
--

SELECT pg_catalog.setval('public.users_id_seq', 15, true);


--
-- Name: cart_items cart_items_cart_id_item_id_key; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.cart_items
    ADD CONSTRAINT cart_items_cart_id_item_id_key UNIQUE (cart_id, item_id);


--
-- Name: cart_items cart_items_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.cart_items
    ADD CONSTRAINT cart_items_pkey PRIMARY KEY (id);


--
-- Name: carts carts_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.carts
    ADD CONSTRAINT carts_pkey PRIMARY KEY (id);


--
-- Name: carts carts_user_id_key; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.carts
    ADD CONSTRAINT carts_user_id_key UNIQUE (user_id);


--
-- Name: idempotency_records idempotency_records_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.idempotency_records
    ADD CONSTRAINT idempotency_records_pkey PRIMARY KEY (key);


--
-- Name: items items_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_pkey PRIMARY KEY (id);


--
-- Name: order_items order_items_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.order_items
    ADD CONSTRAINT order_items_pkey PRIMARY KEY (id);


--
-- Name: orders orders_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (token);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_cart_items_cart_id; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_cart_items_cart_id ON public.cart_items USING btree (cart_id);


--
-- Name: idx_carts_user_id; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_carts_user_id ON public.carts USING btree (user_id);


--
-- Name: idx_items_name; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_items_name ON public.items USING btree (name);


--
-- Name: idx_items_price; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_items_price ON public.items USING btree (price);


--
-- Name: idx_order_items_item_id; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_order_items_item_id ON public.order_items USING btree (item_id);


--
-- Name: idx_order_items_order_id; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_order_items_order_id ON public.order_items USING btree (order_id);


--
-- Name: idx_orders_created_at; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_orders_created_at ON public.orders USING btree (created_at DESC);


--
-- Name: idx_orders_user_id_status; Type: INDEX; Schema: public; Owner: lala
--

CREATE INDEX idx_orders_user_id_status ON public.orders USING btree (user_id, status);


--
-- Name: cart_items cart_items_cart_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.cart_items
    ADD CONSTRAINT cart_items_cart_id_fkey FOREIGN KEY (cart_id) REFERENCES public.carts(id) ON DELETE CASCADE;


--
-- Name: cart_items cart_items_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.cart_items
    ADD CONSTRAINT cart_items_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id);


--
-- Name: order_items order_items_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.order_items
    ADD CONSTRAINT order_items_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id);


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lala
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

\unrestrict nWIiNH9dzvef7sZIMlsW8CAFqO33ft6do3n25aAlFCZ6tEcEi8XnRotqsmpX3iU

