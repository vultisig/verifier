INSERT INTO plugin_policies (id, public_key, plugin_id, plugin_version, policy_version, signature, recipe, active) VALUES 

-- DCA Plugin policy
(
    '20000000-0000-0000-0000-000000000001',
    '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6',
    'vultisig-dca-0000',
    '1.0.0',
    1,
    'test-signature-dca-policy-001',
    'eyJ0eXBlIjoiZGNhIiwicGFyYW1ldGVycyI6eyJhbW91bnQiOiIxMDAiLCJpbnRlcnZhbCI6ImRhaWx5IiwidG9rZW5fYSI6IkVUSCIsInRva2VuX2IiOiJVU0RDIn19',
    true
),

-- Payroll Plugin policy  
(
    '20000000-0000-0000-0000-000000000002',
    '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6',
    'vultisig-payroll-0000', 
    '1.0.0',
    1,
    'test-signature-payroll-policy-001',
    'eyJ0eXBlIjoicGF5cm9sbCIsInBhcmFtZXRlcnMiOnsicGF5X2ZyZXF1ZW5jeSI6Im1vbnRobHkiLCJlbXBsb3llZXMiOlsie1wiYWRkcmVzc1wiOlwiMHgxMjM0XCIsXCJhbW91bnRcIjpcIjUwMDBcIn0iXX19',
    true
) ON CONFLICT (id) DO NOTHING;