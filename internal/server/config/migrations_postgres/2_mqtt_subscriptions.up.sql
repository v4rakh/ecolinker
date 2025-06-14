create table if not exists mqtt_subscriptions
(
    id          uuid                     not null constraint uni_mqtt_subs_id primary key,
    topic_kind  text                     not null,
    device_sn   text                     not null constraint fk_mqtt_subs_device references devices on update cascade on delete cascade,
    created_at  timestamp with time zone not null,
    updated_at  timestamp with time zone not null
);

create unique index idx_d_tk on mqtt_subscriptions (topic_kind, device_sn);

commit;
