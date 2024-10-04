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
            $table->string('commit_sha');
            $table->string('branch')->default('master');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('test_session_services', function (Blueprint $table) {
            $table->dropColumn(['commit_sha', 'branch']);
        });
    }
};
