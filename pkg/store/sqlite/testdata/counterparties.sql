INSERT INTO counterparties (id, source, directory_id, registered_directory, protocol, common_name, endpoint, name, website, country, business_category, vasp_categories, verified_on, ivms101, created, modified, lei) VALUES
    -- ID: "01JXTQCDE6ZES5MPXNW7K19QVQ"
    (x'0197757635c6fbb25a5bb5e1e614df77', "daybreak", '67e4a151-6607-505f-a6ac-55426aa8a677', "daybreak.rotational.io", "sunrise", "daybreak.example.com", "email:compliance@example.com", "Example Daybreak Counterparty", "https://example.com", "US", null, null, null, null, "2024-11-16T15:25:47-10:00", "2024-11-16T15:25:47-10:00", "01234567889abcdef")
;

INSERT INTO contacts (id, name, email, role, counterparty_id, created, modified) VALUES
    -- ID: "01JXTW2Y53KRDB033ZT5P3B007"
    (x'019775c178a39e1ab00c7fd16c358007', "Example Daybreak Compliance Contact", "compliance@daybreak.example.com", "Compliance Contact", x'0197757635c6fbb25a5bb5e1e614df77', "2024-11-16T15:25:48-10:00", "2024-11-16T15:25:48-10:00")
;
