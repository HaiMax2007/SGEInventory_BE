-- =====================================================================
-- SGE Inventory - Schema Sistem Manajemen Barang
-- Engine : InnoDB | Charset : utf8mb4 | Kompatibel MySQL 8 / MariaDB 11
-- =====================================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ---------------------------------------------------------------------
-- lembaga : Lembaga / Divisi / Unit Kerja (BEM, HIMA, UKM X, ...)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS lembaga (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name        VARCHAR(150)    NOT NULL,
    description TEXT            NULL,
    created_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at  DATETIME        NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_lembaga_name (name),
    KEY idx_lembaga_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- users : pengguna sistem (tidak terikat lembaga; lembaga dipilih saat
-- mengajukan peminjaman, lihat borrowings.lembaga_id)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    full_name     VARCHAR(255)    NOT NULL,
    email         VARCHAR(255)    NOT NULL,
    password_hash VARCHAR(255)    NOT NULL,
    contact       VARCHAR(50)     NULL,
    role          ENUM('ADMIN','USER') NOT NULL DEFAULT 'USER',
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at    DATETIME        NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email),
    KEY idx_users_role (role),
    KEY idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- items : master barang / inventaris
-- total_quantity     : jumlah barang yang dimiliki (diatur admin)
-- available_quantity : sisa yang bisa dipinjam (berkurang saat pengajuan
--                      di-ACC, bertambah saat barang dikembalikan)
-- Status tidak disimpan; dihitung dari available_quantity
-- (> 0 = TERSEDIA, 0 = HABIS).
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS items (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name               VARCHAR(255)    NOT NULL,
    total_quantity     INT UNSIGNED    NOT NULL DEFAULT 0,
    available_quantity INT UNSIGNED    NOT NULL DEFAULT 0,
    created_at         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at         DATETIME        NULL,
    PRIMARY KEY (id),
    KEY idx_items_available (available_quantity),
    KEY idx_items_deleted_at (deleted_at),
    CONSTRAINT chk_items_available_lte_total CHECK (available_quantity <= total_quantity)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- borrowings : header pengajuan peminjaman
-- reviewed_by NULL + status REJECTED = ditolak otomatis oleh sistem
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS borrowings (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id             BIGINT UNSIGNED NOT NULL,
    lembaga_id          BIGINT UNSIGNED NOT NULL,
    status              ENUM('PENDING','APPROVED','REJECTED','RETURNED') NOT NULL DEFAULT 'PENDING',
    purpose             VARCHAR(500)    NULL,
    borrow_date         DATE            NOT NULL,
    planned_return_date DATE            NOT NULL,
    actual_return_date  DATE            NULL,
    reviewed_by         BIGINT UNSIGNED NULL,
    reviewed_at         DATETIME        NULL,
    rejection_reason    VARCHAR(500)    NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_borrowings_user_status (user_id, status),
    KEY idx_borrowings_lembaga_id (lembaga_id),
    KEY idx_borrowings_status_created (status, created_at),
    KEY idx_borrowings_reviewed_by (reviewed_by),
    CONSTRAINT chk_borrowings_dates CHECK (planned_return_date >= borrow_date),
    CONSTRAINT fk_borrowings_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_borrowings_lembaga FOREIGN KEY (lembaga_id)
        REFERENCES lembaga (id) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_borrowings_reviewer FOREIGN KEY (reviewed_by)
        REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- borrowing_items : detail barang pada tiap pengajuan (multi-item)
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS borrowing_items (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    borrowing_id BIGINT UNSIGNED NOT NULL,
    item_id      BIGINT UNSIGNED NOT NULL,
    quantity     INT UNSIGNED    NOT NULL,
    status_item  ENUM('PENDING','APPROVED','REJECTED','RETURNED') NOT NULL DEFAULT 'PENDING',
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_borrowing_items_borrowing_item (borrowing_id, item_id),
    KEY idx_borrowing_items_item_status (item_id, status_item),
    CONSTRAINT chk_borrowing_items_quantity CHECK (quantity > 0),
    CONSTRAINT fk_borrowing_items_borrowing FOREIGN KEY (borrowing_id)
        REFERENCES borrowings (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_borrowing_items_item FOREIGN KEY (item_id)
        REFERENCES items (id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- item_consignments : penitipan barang milik user ke organisasi
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS item_consignments (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    owner_user_id   BIGINT UNSIGNED NOT NULL,
    item_name       VARCHAR(255)    NOT NULL,
    quantity        INT UNSIGNED    NOT NULL,
    start_date      DATE            NOT NULL,
    end_date        DATE            NOT NULL,
    status          ENUM('DITITIPKAN','DIAMBIL','EXPIRED') NOT NULL DEFAULT 'DITITIPKAN',
    condition_notes TEXT            NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_consignments_owner (owner_user_id),
    KEY idx_consignments_status_end (status, end_date),
    CONSTRAINT chk_consignments_quantity CHECK (quantity > 0),
    CONSTRAINT chk_consignments_dates CHECK (end_date >= start_date),
    CONSTRAINT fk_consignments_owner FOREIGN KEY (owner_user_id)
        REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;

-- =====================================================================
-- Data awal (khusus development)
-- password_hash adalah hash bcrypt; password aslinya tidak disimpan di sini.
-- INSERT IGNORE: tidak error jika email/id sudah ada.
-- =====================================================================
INSERT IGNORE INTO users (id, full_name, email, password_hash, contact, role, created_at, updated_at)
VALUES (1, 'Max', 'marshallmaranda34@gmail.com',
        '$2a$10$S7QOM2wmv727X/rQo.okB.ta0isdJztH63bqwxcHYeTI5.p1tfX6.',
        '089602160276', 'ADMIN', '2026-11-12 10:01:38', '2026-11-12 10:01:38');
