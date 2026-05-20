-- 068_populate_missing_categories_brands.sql
-- Idempotent: inserts only missing categories, subcategories, manufacturers and brands
-- Safe to re-run multiple times

BEGIN;

-- Categories (by abbreviation)
INSERT INTO categories (name, abbreviation)
SELECT v.name, v.abbreviation
FROM (VALUES
('Lighting', 'LT'),
('Audio', 'AU'),
('Video', 'VI'),
('Power Distribution', 'PW'),
('Cables & Connectors', 'CA'),
('Rigging & Staging', 'RS'),
('Backline', 'BL'),
('Communication', 'CM'),
('Accessories', 'AC'),
('ICT & Network', 'ICT')
) AS v(name, abbreviation)
WHERE NOT EXISTS (SELECT 1 FROM categories c WHERE c.abbreviation = v.abbreviation);

-- Subcategories (idempotent)
INSERT INTO subcategories (subcategoryid, name, abbreviation, categoryid)
SELECT v.subcategoryid, v.name, v.abbreviation, (SELECT MIN(categoryid) FROM categories WHERE abbreviation = v.cat_abbr)
FROM (VALUES
('LT-MH','Moving Heads','MH','LT'),
('LT-PAR','LED Pars','PAR','LT'),
('LT-BAR','LED Bars & Strips','BAR','LT'),
('LT-FSP','Followspots','FSP','LT'),
('LT-HAZ','Hazers & Fog Machines','HAZ','LT'),
('LT-STB','Strobes','STB','LT'),
('LT-CTL','Controllers & Dimmers','CTL','LT'),
('AU-MIC','Microphones','MIC','AU'),
('AU-SPA','PA Speakers','SPA','AU'),
('AU-MON','Stage Monitors','MON','AU'),
('AU-AMP','Amplifiers','AMP','AU'),
('AU-MIX','Mixing Consoles','MIX','AU'),
('AU-WRL','Wireless Systems','WRL','AU'),
('AU-IEM','In-Ear Monitors','IEM','AU'),
('AU-LAR','Line Arrays','LAR','AU'),
('VI-PRJ','Projectors','PRJ','VI'),
('VI-LED','LED Screens','LED','VI'),
('VI-MNT','Monitors & Displays','MNT','VI'),
('VI-SWT','Video Switchers','SWT','VI'),
('VI-CAM','Cameras','CAM','VI'),
('VI-MSV','Media Servers','MSV','VI'),
('PW-DST','Distribution Boards','DST','PW'),
('PW-GEN','Generators','GEN','PW'),
('PW-UPS','UPS Systems','UPS','PW'),
('PW-RLS','Cable Reels','RLS','PW'),
('CA-PWC','Power Cables','PWC','CA'),
('CA-AUC','Audio Cables','AUC','CA'),
('CA-VIC','Video Cables','VIC','CA'),
('CA-DMX','DMX Cables','DMX','CA'),
('CA-NTC','Network Cables','NTC','CA'),
('CA-MCC','Multicore','MCC','CA'),
('RS-TRS','Trussing','TRS','RS'),
('RS-CHH','Chain Hoists','CHH','RS'),
('RS-STA','Stands & Tripods','STA','RS'),
('RS-PLT','Staging Platforms','PLT','RS'),
('RS-DRP','Drapes & Fabric','DRP','RS'),
('BL-GTR','Guitars','GTR','BL'),
('BL-KEY','Keyboards & Synths','KEY','BL'),
('BL-DRM','Drums & Percussion','DRM','BL'),
('BL-AMP','Guitar & Bass Amps','AMP','BL'),
('CM-INT','Intercoms','INT','CM'),
('CM-WLK','Walkie-Talkies','WLK','CM'),
('AC-CAS','Cases & Flight Cases','CAS','AC'),
('AC-HWT','Hardware & Tools','HWT','AC'),
('AC-ADP','Adapters & Connectors','ADP','AC'),
('ICT-NSW','Network Switches','NSW','ICT'),
('ICT-RAP','Routers & Access Points','RAP','ICT'),
('ICT-SRV','Servers & Compute','SRV','ICT')
) AS v(subcategoryid, name, abbreviation, cat_abbr)
WHERE NOT EXISTS (SELECT 1 FROM subcategories s WHERE s.subcategoryid = v.subcategoryid)
  AND EXISTS (SELECT 1 FROM categories c WHERE c.abbreviation = v.cat_abbr);

-- Manufacturers
INSERT INTO manufacturer (name, website)
SELECT v.name, v.website
FROM (VALUES
('Shure',                  'https://www.shure.com'),
('Sennheiser',             'https://www.sennheiser.com'),
('d&b audiotechnik',       'https://www.dbaudio.com'),
('L-Acoustics',            'https://www.l-acoustics.com'),
('JBL Professional',       'https://jblpro.com'),
('Yamaha',                 'https://usa.yamaha.com'),
('QSC',                    'https://www.qsc.com'),
('Crown International',    'https://www.crownaudio.com'),
('Martin by Harman',       'https://www.martin.com'),
('Robe Lighting',          'https://www.robe.cz'),
('Chauvet Professional',   'https://www.chauvetprofessional.com'),
('ETC',                    'https://www.etcconnect.com'),
('GLP',                    'https://www.glp.de'),
('Ayrton',                 'https://www.ayrton.eu'),
('Claypaky',               'https://www.claypaky.it'),
('Elation Professional',   'https://www.elationlighting.com'),
('DiGiCo',                 'https://www.digico.biz'),
('Midas',                  'https://www.midasconsoles.com'),
('Allen & Heath',          'https://www.allen-heath.com'),
('Roland',                 'https://www.roland.com'),
('Blackmagic Design',      'https://www.blackmagicdesign.com'),
('Panasonic',              'https://www.panasonic.com'),
('Sony',                   'https://pro.sony'),
('Christie',               'https://www.christiedigital.com'),
('Barco',                  'https://www.barco.com'),
('Prolyte Group',          'https://www.prolyte.com'),
('Global Truss',           'https://www.global-truss.com'),
('Neutrik',                'https://www.neutrik.com'),
('Amphenol',               'https://www.amphenol.com'),
('Obsidian Control Systems','https://www.obsidiancontrol.com'),
('MA Lighting',            'https://www.malighting.com'),
('Avolites',               'https://www.avolites.com')
) AS v(name, website)
WHERE NOT EXISTS (SELECT 1 FROM manufacturer m WHERE m.name = v.name);

-- Brands (link to manufacturer)
INSERT INTO brands (name, manufacturerid)
SELECT b.name, m.manufacturerid
FROM (VALUES
('Shure','Shure'),
('Sennheiser','Sennheiser'),
('Neumann','Sennheiser'),
('d&b audiotechnik','d&b audiotechnik'),
('L-Acoustics','L-Acoustics'),
('JBL Professional','JBL Professional'),
('JBL','JBL Professional'),
('Yamaha','Yamaha'),
('Nexo','Yamaha'),
('QSC','QSC'),
('Crown','Crown International'),
('Martin','Martin by Harman'),
('Robe','Robe Lighting'),
('Chauvet Professional','Chauvet Professional'),
('Chauvet DJ','Chauvet Professional'),
('ETC','ETC'),
('GLP','GLP'),
('Ayrton','Ayrton'),
('Claypaky','Claypaky'),
('Elation','Elation Professional'),
('DiGiCo','DiGiCo'),
('Midas','Midas'),
('Allen & Heath','Allen & Heath'),
('Roland','Roland'),
('BOSS','Roland'),
('Blackmagic Design','Blackmagic Design'),
('ATEM','Blackmagic Design'),
('Panasonic','Panasonic'),
('Sony','Sony'),
('Christie','Christie'),
('Barco','Barco'),
('Prolyte','Prolyte Group'),
('Global Truss','Global Truss'),
('Neutrik','Neutrik'),
('Amphenol','Amphenol'),
('Onyx','Obsidian Control Systems'),
('grandMA','MA Lighting'),
('Avolites','Avolites')
) AS b(name, mname)
JOIN manufacturer m ON m.name = b.mname
WHERE NOT EXISTS (SELECT 1 FROM brands br WHERE br.name = b.name);

COMMIT;
