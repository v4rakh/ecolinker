create table if not exists devices
(
    sn text     not null constraint uni_devices_sn primary key,
    kind text   not null,
    label text  not null,
    created_at  timestamp with time zone not null,
    updated_at  timestamp with time zone not null
);

commit;
