create table if not exists collectors
(
    id              uuid                     not null constraint uni_collectors_id primary key,
    kind            text                     not null,
    frequency       text                     not null,
    device_sn       text                     not null constraint fk_collectors_device references devices on update cascade on delete cascade,
    payload         jsonb                    not null,
    created_at      timestamp with time zone not null,
    updated_at      timestamp with time zone not null
);

commit;
