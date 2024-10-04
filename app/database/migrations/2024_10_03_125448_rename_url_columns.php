<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('test_session_services', function (Blueprint $table) {
            $table->renameColumn('url', 'path');
        });

        Schema::table('test_session_service_suites', function (Blueprint $table) {
            $table->renameColumn('url', 'path');
        });

        Schema::table('test_session_service_suite_tests', function (Blueprint $table) {
            $table->renameColumn('url', 'path');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('test_session_services', function (Blueprint $table) {
            $table->renameColumn('path', 'url');
        });

        Schema::table('test_session_service_suites', function (Blueprint $table) {
            $table->renameColumn('path', 'url');
        });

        Schema::table('test_session_service_suite_tests', function (Blueprint $table) {
            $table->renameColumn('path', 'url');
        });
    }
};
